package websocket

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/errors"
)

// websocketSendBufferSize is the number of elements the send channel
// can queue before blocking.  Note that this only applies to requests
// handled directly in the websocket client input handler or the async
// handler since notifications have their own queuing mechanism
// independent of the send channel buffer.
const (
	logModule               = "websocket"
	websocketSendBufferSize = 50
)

var (
	// ErrWSParse means a request parsing error
	ErrWSParse = errors.New("Websocket request parsing error")
	// ErrWSInternal means service handling errors
	ErrWSInternal = errors.New("Websocket Internal error")
	// ErrWSClientQuit means the websocket client is disconnected
	ErrWSClientQuit = errors.New("Websocket client quit")

	// timeZeroVal is simply the zero value for a time.Time and is used to avoid creating multiple instances.
	timeZeroVal time.Time
)

type semaphore chan struct{}

func makeSemaphore(n int) semaphore {
	return make(chan struct{}, n)
}

func (s semaphore) acquire() { s <- struct{}{} }
func (s semaphore) release() { <-s }

// wsTopicHandler describes a callback function used to handle a specific topic.
type wsTopicHandler func(*WSClient)

// wsHandlers maps websocket topic strings to appropriate websocket handler
// functions.  This is set by init because help references wsHandlers and thus
// causes a dependency loop.
var wsHandlers = map[string]wsTopicHandler{
	"notify_raw_blocks":            handleNotifyBlocks,
	"notify_new_transactions":      handleNotifyNewTransactions,
	"stop_notify_raw_blocks":       handleStopNotifyBlocks,
	"stop_notify_new_transactions": handleStopNotifyNewTransactions,
}

// responseMessage houses a message to send to a connected websocket client as
// well as a channel to reply on when the message is sent.
type responseMessage struct {
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
	conn *websocket.Conn
	// disconnected indicated whether or not the websocket client is disconnected.
	disconnected bool
	// addr is the remote address of the client.
	addr              string
	serviceRequestSem semaphore
	ntfnChan          chan []byte
	sendChan          chan responseMessage
	quit              chan struct{}
	wg                sync.WaitGroup
	notificationMgr   *WSNotificationManager
}

// NewWebsocketClient means to create a new object to the connected websocket client
func NewWebsocketClient(w http.ResponseWriter, r *http.Request, notificationMgr *WSNotificationManager) (*WSClient, error) {
	// Limit max number of websocket clients.
	if notificationMgr.IsMaxConnect() {
		return nil, fmt.Errorf("numOfMaxWS: %d, disconnecting: %s", notificationMgr.MaxNumWebsockets, r.RemoteAddr)
	}

	// Attempt to upgrade the connection to a websocket connection using the default size for read/write buffers.
	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	if err != nil {
		return nil, err
	}

	conn.SetReadDeadline(timeZeroVal)

	client := &WSClient{
		conn:              conn,
		addr:              r.RemoteAddr,
		serviceRequestSem: makeSemaphore(notificationMgr.maxNumConcurrentReqs),
		ntfnChan:          make(chan []byte, 1), // nonblocking sync
		sendChan:          make(chan responseMessage, websocketSendBufferSize),
		quit:              make(chan struct{}),
		notificationMgr:   notificationMgr,
	}
	return client, nil
}

// inHandler handles all incoming messages for the websocket connection.
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
			if err != io.EOF {
				log.WithFields(log.Fields{"module": logModule, "remoteAddress": c.addr, "error": err}).Error("Websocket receive error")
			}
			break out
		}

		var request WSRequest
		if err = json.Unmarshal(msg, &request); err != nil {
			respError := errors.Wrap(err, ErrWSParse)
			resp := NewWSResponse(NTRequestStatus.String(), nil, respError)
			reply, err := json.Marshal(resp)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "error": err}).Error("Failed to marshal parse failure reply")
				continue
			}

			c.SendMessage(reply, nil)
			continue
		}

		c.serviceRequestSem.acquire()
		go func() {
			c.serviceRequest(request.Topic)
			c.serviceRequestSem.release()
		}()
	}

	// Ensure the connection is closed.
	c.Disconnect()
	c.wg.Done()
	log.WithFields(log.Fields{"module": logModule, "remoteAddress": c.addr}).Debug("Websocket client input handler done")
}

func (c *WSClient) serviceRequest(topic string) {
	var respErr error

	if wsHandler, ok := wsHandlers[topic]; ok {
		wsHandler(c)
	} else {
		err := fmt.Errorf("There is not this topic: %s", topic)
		respErr = errors.Wrap(err, ErrWSInternal)
		log.WithFields(log.Fields{"module": logModule, "topic": topic}).Debug("There is not this topic")
	}

	resp := NewWSResponse(NTRequestStatus.String(), nil, respErr)
	reply, err := json.Marshal(resp)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Debug("Failed to marshal parse failure reply")
		return
	}

	c.SendMessage(reply, nil)
}

// notificationQueueHandler handles the queuing of outgoing notifications for  the websocket client.
func (c *WSClient) notificationQueueHandler() {
	ntfnSentChan := make(chan bool, 1) // nonblocking sync

	// pendingNtfns is used as a queue for notifications that are ready to
	// be sent once there are no outstanding notifications currently being
	// sent.
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
		// This channel is notified when a notification has been sent across the network socket.
		case <-ntfnSentChan:
			// No longer waiting if there are no more messages in the pending messages queue.
			next := pendingNtfns.Front()
			if next == nil {
				waiting = false
				continue
			}

			// Notify the outHandler about the next item to asynchronously send.
			msg := pendingNtfns.Remove(next).([]byte)
			c.SendMessage(msg, ntfnSentChan)
		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before exiting so nothing is left waiting around to send.
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
	log.WithFields(log.Fields{"module": logModule, "remoteAddress": c.addr}).Debug("Websocket client notification queue handler done")
}

// outHandler handles all outgoing messages for the websocket connection.
func (c *WSClient) outHandler() {
out:
	for {
		// Send any messages ready for send until the quit channel is closed.
		select {
		case r := <-c.sendChan:
			if err := c.conn.WriteMessage(websocket.TextMessage, r.msg); err != nil {
				log.WithFields(log.Fields{"module": logModule, "error": err}).Error("Failed to send message to wesocket client")
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

	// Drain any wait channels before exiting so nothing is left waiting around to send.
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
	log.WithFields(log.Fields{"module": logModule, "remoteAddress": c.addr}).Debug("Websocket client output handler done")
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

	c.sendChan <- responseMessage{msg: marshalledJSON, doneChan: doneChan}
}

// QueueNotification queues the passed notification to be sent to the websocket client.
func (c *WSClient) QueueNotification(marshalledJSON []byte) error {
	// Don't queue the message if disconnected.
	if c.Disconnected() {
		return ErrWSClientQuit
	}

	c.ntfnChan <- marshalledJSON
	return nil
}

// Disconnected returns whether or not the websocket client is disconnected.
func (c *WSClient) Disconnected() bool {
	c.Lock()
	defer c.Unlock()

	return c.disconnected
}

// Disconnect disconnects the websocket client.
func (c *WSClient) Disconnect() {
	c.Lock()
	defer c.Unlock()

	// Nothing to do if already disconnected.
	if c.disconnected {
		return
	}

	log.WithFields(log.Fields{"module": logModule, "remoteAddress": c.addr}).Info("Disconnecting websocket client")

	close(c.quit)
	c.conn.Close()
	c.disconnected = true
}

// Start begins processing input and output messages.
func (c *WSClient) Start() {
	log.WithFields(log.Fields{"module": logModule, "remoteAddress": c.addr}).Info("Starting websocket client")

	c.wg.Add(3)
	go c.inHandler()
	go c.notificationQueueHandler()
	go c.outHandler()
}

// WaitForShutdown blocks until the websocket client goroutines are stopped and the connection is closed.
func (c *WSClient) WaitForShutdown() {
	c.wg.Wait()
}

// handleNotifyBlocks implements the notifyblocks topic extension for websocket connections.
func handleNotifyBlocks(wsc *WSClient) {
	wsc.notificationMgr.RegisterBlockUpdates(wsc)
}

// handleStopNotifyBlocks implements the stopnotifyblocks topic extension for websocket connections.
func handleStopNotifyBlocks(wsc *WSClient) {
	wsc.notificationMgr.UnregisterBlockUpdates(wsc)
}

// handleNotifyNewTransations implements the notifynewtransactions topic extension for websocket connections.
func handleNotifyNewTransactions(wsc *WSClient) {
	wsc.notificationMgr.RegisterNewMempoolTxsUpdates(wsc)
}

// handleStopNotifyNewTransations implements the stopnotifynewtransactions topic extension for websocket connections.
func handleStopNotifyNewTransactions(wsc *WSClient) {
	wsc.notificationMgr.UnregisterNewMempoolTxsUpdates(wsc)
}
