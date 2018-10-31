package websocket

import (
	"container/list"
	"encoding/json"
	"io"
	"sync"

	"github.com/bytom/errors"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	// websocketSendBufferSize is the number of elements the send channel
	// can queue before blocking.  Note that this only applies to requests
	// handled directly in the websocket client input handler or the async
	// handler since notifications have their own queuing mechanism
	// independent of the send channel buffer.
	websocketSendBufferSize = 50
)

type semaphore chan struct{}

func makeSemaphore(n int) semaphore {
	return make(chan struct{}, n)
}

func (s semaphore) acquire() { s <- struct{}{} }
func (s semaphore) release() { <-s }

// wsCommandHandler describes a callback function used to handle a specific
// command.
type wsCommandHandler func(*WSClient, interface{}) (interface{}, error)

// wsHandlers maps websocket command strings to appropriate websocket handler
// functions.  This is set by init because help references wsHandlers and thus
// causes a dependency loop.
var wsHandlers map[string]wsCommandHandler
var wsHandlersBeforeInit = map[string]wsCommandHandler{
	"notifyblocks":              handleNotifyBlocks,
	"notifynewtransactions":     handleNotifyNewTransactions,
	"stopnotifyblocks":          handleStopNotifyBlocks,
	"stopnotifynewtransactions": handleStopNotifyNewTransactions,
}

// newWebsocketClient returns a new websocket client given the notification
// manager, websocket connection, remote address, and whether or not the client
// has already been authenticated (via HTTP Basic access authentication).  The
// returned client is ready to start.  Once started, the client will process
// incoming and outgoing messages in separate goroutines complete with queuing
// and asynchrous handling for long-running operations.
func NewWebsocketClient(conn *websocket.Conn, remoteAddr string, ntfnMgr *WSNotificationManager, maxConcurrentReqs int) (*WSClient, error) {

	client := &WSClient{
		conn:              conn,
		addr:              remoteAddr,
		serviceRequestSem: makeSemaphore(maxConcurrentReqs),
		ntfnChan:          make(chan []byte, 1), // nonblocking sync
		sendChan:          make(chan wsResponse, websocketSendBufferSize),
		quit:              make(chan struct{}),
		ntfnMgr:           ntfnMgr,
	}
	return client, nil
}

// wsResponse houses a message to send to a connected websocket client as
// well as a channel to reply on when the message is sent.
type wsResponse struct {
	msg      []byte
	doneChan chan bool
}

// WSClient provides an abstraction for handling a websocket client.  The
// overall data flow is split into 3 main goroutines, a possible 4th goroutine
// for long-running operations (only started if request is made), and a
// websocket manager which is used to allow things such as broadcasting
// requested notifications to all connected websocket clients.   Inbound
// messages are read via the inHandler goroutine and generally dispatched to
// their own handler.  However, certain potentially long-running operations such
// as rescans, are sent to the asyncHander goroutine and are limited to one at a
// time.  There are two outbound message types - one for responding to client
// requests and another for async notifications.  Responses to client requests
// use SendMessage which employs a buffered channel thereby limiting the number
// of outstanding requests that can be made.  Notifications are sent via
// QueueNotification which implements a queue via notificationQueueHandler to
// ensure sending notifications from other subsystems can't block.  Ultimately,
// all messages are sent via the outHandler.
type WSClient struct {
	sync.Mutex
	// conn is the underlying websocket connection.
	conn *websocket.Conn

	// disconnected indicated whether or not the websocket client is
	// disconnected.
	disconnected bool

	// addr is the remote address of the client.
	addr string

	// Networking infrastructure.
	serviceRequestSem semaphore
	ntfnChan          chan []byte
	sendChan          chan wsResponse
	quit              chan struct{}
	wg                sync.WaitGroup
	ntfnMgr           *WSNotificationManager
}

// inHandler handles all incoming messages for the websocket connection.  It
// must be run as a goroutine.
func (c *WSClient) inHandler() {
out:
	for {
		// Break out of the loop once the quit channel has been closed.
		// Use a non-blocking select here so we fall through otherwise.
		select {
		case <-c.quit:
			break out
		default:
		}
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			// Log the error if it's not due to disconnecting.
			if err != io.EOF {
				log.Errorf("Websocket receive error from %s: %v", c.addr, err)

			}
			break out
		}
		var request WSRequest
		err = json.Unmarshal(msg, &request)
		if err != nil {

			jsonErr := &WSError{
				Code:    ErrWSParse.Code,
				Message: "Failed to parse request: " + err.Error(),
			}
			reply, err := MarshalResponse(nil, nil, jsonErr)
			if err != nil {
				log.Errorf("Failed to marshal parse failure reply: %v", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}
		c.serviceRequestSem.acquire()
		go func() {
			c.serviceRequest(request.Method)
			c.serviceRequestSem.release()
		}()
	}

	// Ensure the connection is closed.
	c.Disconnect()
	c.wg.Done()
	log.Debugf("Websocket client input handler done for %s", c.addr)
}

func (c *WSClient) serviceRequest(method string) {
	var (
		result interface{}
		err    error
	)
	wsHandler, ok := wsHandlers[method]
	if ok {
		result, err = wsHandler(c, method)
	} else {
		log.Errorf("There is not this method: %s", method)
		return
	}
	var jsonErr *WSError
	if err != nil {
		if jErr, ok := err.(*WSError); ok {
			jsonErr = jErr
		} else {
			jsonErr = NewWSError(ErrWSInternal.Code, err.Error())
		}
	}

	reply, err := MarshalResponse(nil, result, jsonErr)
	if err != nil {
		log.Errorf("Failed to marshal parse failure reply: %v", err)
		return
	}
	c.SendMessage(reply, nil)
}

// notificationQueueHandler handles the queuing of outgoing notifications for
// the websocket client.  This runs as a muxer for various sources of input to
// ensure that queuing up notifications to be sent will not block.  Otherwise,
// slow clients could bog down the other systems (such as the mempool or block
// manager) which are queuing the data.  The data is passed on to outHandler to
// actually be written.  It must be run as a goroutine.
func (c *WSClient) notificationQueueHandler() {
	ntfnSentChan := make(chan bool, 1) // nonblocking sync

	// pendingNtfns is used as a queue for notifications that are ready to
	// be sent once there are no outstanding notifications currently being
	// sent.  The waiting flag is used over simply checking for items in the
	// pending list to ensure cleanup knows what has and hasn't been sent
	// to the outHandler.  Currently no special cleanup is needed, however
	// if something like a done channel is added to notifications in the
	// future, not knowing what has and hasn't been sent to the outHandler
	// (and thus who should respond to the done channel) would be
	// problematic without using this approach.
	pendingNtfns := list.New()
	waiting := false
out:
	for {
		select {
		// This channel is notified when a message is being queued to
		// be sent across the network socket.  It will either send the
		// message immediately if a send is not already in progress, or
		// queue the message to be sent once the other pending messages
		// are sent.
		case msg := <-c.ntfnChan:
			if !waiting {
				c.SendMessage(msg, ntfnSentChan)
			} else {
				pendingNtfns.PushBack(msg)
			}
			waiting = true

		// This channel is notified when a notification has been sent
		// across the network socket.
		case <-ntfnSentChan:
			// No longer waiting if there are no more messages in
			// the pending messages queue.
			next := pendingNtfns.Front()
			if next == nil {
				waiting = false
				continue
			}

			// Notify the outHandler about the next item to
			// asynchronously send.
			msg := pendingNtfns.Remove(next).([]byte)
			c.SendMessage(msg, ntfnSentChan)

		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before exiting so nothing is left waiting
	// around to send.
cleanup:
	for {
		select {
		case <-c.ntfnChan:
		case <-ntfnSentChan:
		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Infof("Websocket client notification queue handler done for %s", c.addr)
}

// outHandler handles all outgoing messages for the websocket connection.  It
// must be run as a goroutine.  It uses a buffered channel to serialize output
// messages while allowing the sender to continue running asynchronously.  It
// must be run as a goroutine.
func (c *WSClient) outHandler() {
out:
	for {
		// Send any messages ready for send until the quit channel is
		// closed.
		select {
		case r := <-c.sendChan:
			err := c.conn.WriteMessage(websocket.TextMessage, r.msg)
			if err != nil {
				c.Disconnect()
				break out
			}
			if r.doneChan != nil {
				r.doneChan <- true
			}

		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before exiting so nothing is left waiting
	// around to send.
cleanup:
	for {
		select {
		case r := <-c.sendChan:
			if r.doneChan != nil {
				r.doneChan <- false
			}
		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Infof("Websocket client output handler done for %s", c.addr)
}

// SendMessage sends the passed json to the websocket client.  It is backed
// by a buffered channel, so it will not block until the send channel is full.
// Note however that QueueNotification must be used for sending async
// notifications instead of the this function.  This approach allows a limit to
// the number of outstanding requests a client can make without preventing or
// blocking on async notifications.
func (c *WSClient) SendMessage(marshalledJSON []byte, doneChan chan bool) {
	// Don't send the message if disconnected.
	if c.Disconnected() {
		if doneChan != nil {
			doneChan <- false
		}
		return
	}

	c.sendChan <- wsResponse{msg: marshalledJSON, doneChan: doneChan}
}

// ErrClientQuit describes the error where a client send is not processed due
// to the client having already been disconnected or dropped.
var ErrClientQuit = errors.New("client quit")

// QueueNotification queues the passed notification to be sent to the websocket
// client.  This function, as the name implies, is only intended for
// notifications since it has additional logic to prevent other subsystems, such
// as the memory pool and block manager, from blocking even when the send
// channel is full.
//
// If the client is in the process of shutting down, this function returns
// ErrClientQuit.  This is intended to be checked by long-running notification
// handlers to stop processing if there is no more work needed to be done.
func (c *WSClient) QueueNotification(marshalledJSON []byte) error {
	// Don't queue the message if disconnected.
	if c.Disconnected() {
		return ErrClientQuit
	}

	c.ntfnChan <- marshalledJSON
	return nil
}

// Disconnected returns whether or not the websocket client is disconnected.
func (c *WSClient) Disconnected() bool {
	c.Lock()
	isDisconnected := c.disconnected
	c.Unlock()

	return isDisconnected
}

// Disconnect disconnects the websocket client.
func (c *WSClient) Disconnect() {
	c.Lock()
	defer c.Unlock()

	// Nothing to do if already disconnected.
	if c.disconnected {
		return
	}

	log.Infof("Disconnecting websocket client %s", c.addr)
	close(c.quit)
	c.conn.Close()
	c.disconnected = true
}

// Start begins processing input and output messages.
func (c *WSClient) Start() {
	log.Infof("Starting websocket client %s", c.addr)

	// Start processing input and output.
	c.wg.Add(3)
	go c.inHandler()
	go c.notificationQueueHandler()
	go c.outHandler()
}

// WaitForShutdown blocks until the websocket client goroutines are stopped
// and the connection is closed.
func (c *WSClient) WaitForShutdown() {
	c.wg.Wait()
}

// handleNotifyBlocks implements the notifyblocks command extension for
// websocket connections.
func handleNotifyBlocks(wsc *WSClient, icmd interface{}) (interface{}, error) {
	wsc.ntfnMgr.RegisterBlockUpdates(wsc)
	return nil, nil
}

// handleStopNotifyBlocks implements the stopnotifyblocks command extension for
// websocket connections.
func handleStopNotifyBlocks(wsc *WSClient, icmd interface{}) (interface{}, error) {
	wsc.ntfnMgr.UnregisterBlockUpdates(wsc)
	return nil, nil
}

// handleNotifyNewTransations implements the notifynewtransactions command
// extension for websocket connections.
func handleNotifyNewTransactions(wsc *WSClient, icmd interface{}) (interface{}, error) {
	wsc.ntfnMgr.RegisterNewMempoolTxsUpdates(wsc)
	return nil, nil
}

// handleStopNotifyNewTransations implements the stopnotifynewtransactions
// command extension for websocket connections.
func handleStopNotifyNewTransactions(wsc *WSClient, icmd interface{}) (interface{}, error) {
	wsc.ntfnMgr.UnregisterNewMempoolTxsUpdates(wsc)
	return nil, nil
}

func init() {
	wsHandlers = wsHandlersBeforeInit
}
