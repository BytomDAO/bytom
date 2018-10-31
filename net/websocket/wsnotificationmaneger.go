package websocket

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/bytom/protocol/bc/types"
	log "github.com/sirupsen/logrus"
)

// Notification types
type notificationBlockConnected types.Block
type notificationBlockDisconnected types.Block
type notificationTxAcceptedByMempool struct {
	isNew bool
	tx    *types.Tx
}

// Notification control requests
type notificationRegisterClient WSClient
type notificationUnregisterClient WSClient
type notificationRegisterBlocks WSClient
type notificationUnregisterBlocks WSClient
type notificationRegisterNewMempoolTxs WSClient
type notificationUnregisterNewMempoolTxs WSClient

type notificationRegisterAddr struct {
	wsc   *WSClient
	addrs []string
}
type notificationUnregisterAddr struct {
	wsc  *WSClient
	addr string
}

// NotificationType represents the type of a notification message.
type NotificationType int

// NotificationCallback is used for a caller to provide a callback for
// notifications about various chain events.
type NotificationCallback func(*Notification)

// Constants for the type of a notification message.
const (

	// NTBlockConnected indicates the associated block was connected to the
	// main chain.
	NTBlockConnected NotificationType = iota

	// NTBlockDisconnected indicates the associated block was disconnected
	// from the main chain.
	NTBlockDisconnected
)

// notificationTypeStrings is a map of notification types back to their constant
// names for pretty printing.
var notificationTypeStrings = map[NotificationType]string{
	NTBlockConnected:    "NTBlockConnected",
	NTBlockDisconnected: "NTBlockDisconnected",
}

// String returns the NotificationType in human-readable form.
func (n NotificationType) String() string {
	if s, ok := notificationTypeStrings[n]; ok {
		return s
	}
	return fmt.Sprintf("Unknown Notification Type (%d)", int(n))
}

type Notification struct {
	Type NotificationType
	Data interface{}
}

// WSNotificationManager is a connection and notification manager used for
// websockets.  It allows websocket clients to register for notifications they
// are interested in.  When an event happens elsewhere in the code such as
// transactions being added to the memory pool or block connects/disconnects,
// the notification manager is provided with the relevant details needed to
// figure out which websocket clients need to be notified based on what they
// have registered for and notifies them accordingly.  It is also used to keep
// track of all connected websocket clients.
type WSNotificationManager struct {
	// queueNotification queues a notification for handling.
	queueNotification chan interface{}

	// notificationMsgs feeds notificationHandler with notifications
	// and client (un)registeration requests from a queue as well as
	// registeration and unregisteration requests from clients.
	notificationMsgs chan interface{}

	// Access channel for current number of connected clients.
	numClients chan int

	// Shutdown handling
	wg   sync.WaitGroup
	quit chan struct{}
}

// queueHandler manages a queue of empty interfaces, reading from in and
// sending the oldest unsent to out.  This handler stops when either of the
// in or quit channels are closed, and closes out before returning, without
// waiting to send any variables still remaining in the queue.
func queueHandler(in <-chan interface{}, out chan<- interface{}, quit <-chan struct{}) {
	var q []interface{}
	var dequeue chan<- interface{}
	skipQueue := out
	var next interface{}
out:
	for {
		select {
		case n, ok := <-in:
			if !ok {
				// Sender closed input channel.
				break out
			}

			// Either send to out immediately if skipQueue is
			// non-nil (queue is empty) and reader is ready,
			// or append to the queue and send later.
			select {
			case skipQueue <- n:
			default:
				q = append(q, n)
				dequeue = out
				skipQueue = nil
				next = q[0]
			}

		case dequeue <- next:
			copy(q, q[1:])
			q[len(q)-1] = nil // avoid leak
			q = q[:len(q)-1]
			if len(q) == 0 {
				dequeue = nil
				skipQueue = out
			} else {
				next = q[0]
			}

		case <-quit:
			break out
		}
	}
	close(out)
}

func (m *WSNotificationManager) SendNotification(typ NotificationType, data interface{}) {
	switch typ {
	case NTBlockConnected:
		block, ok := data.(*types.Block)
		if !ok {
			log.Errorf("Chain connected notification is not a block.")
			break
		}

		// Notify registered websocket clients of incoming block.
		m.NotifyBlockConnected(block)
	case NTBlockDisconnected:
		block, ok := data.(*types.Block)
		if !ok {
			log.Errorf("Chain disconnected notification is not a block.")
			break
		}

		// Notify registered websocket clients.
		m.NotifyBlockDisconnected(block)
	}
}

// queueHandler maintains a queue of notifications and notification handler
// control messages.
func (m *WSNotificationManager) queueHandler() {
	queueHandler(m.queueNotification, m.notificationMsgs, m.quit)
	m.wg.Done()
}

// NotifyBlockConnected passes a block newly-connected to the best chain
// to the notification manager for block and transaction notification
// processing.
func (m *WSNotificationManager) NotifyBlockConnected(block *types.Block) {
	select {
	case m.queueNotification <- (*notificationBlockConnected)(block):
	case <-m.quit:
	}
}

// NotifyBlockDisconnected passes a block disconnected from the best chain
// to the notification manager for block notification processing.
func (m *WSNotificationManager) NotifyBlockDisconnected(block *types.Block) {
	select {
	case m.queueNotification <- (*notificationBlockDisconnected)(block):
	case <-m.quit:
	}
}

// NotifyMempoolTx passes a transaction accepted by mempool to the
// notification manager for transaction notification processing.  If
// isNew is true, the tx is is a new transaction, rather than one
// added to the mempool during a reorg.
func (m *WSNotificationManager) NotifyMempoolTx(tx *types.Tx, isNew bool) {
	n := &notificationTxAcceptedByMempool{
		isNew: isNew,
		tx:    tx,
	}

	select {
	case m.queueNotification <- n:
	case <-m.quit:
	}
}

// notificationHandler reads notifications and control messages from the queue
// handler and processes one at a time.
func (m *WSNotificationManager) notificationHandler() {
	// clients is a map of all currently connected websocket clients.
	clients := make(map[chan struct{}]*WSClient)

	// Maps used to hold lists of websocket clients to be notified on
	// certain events.  Each websocket client also keeps maps for the events
	// which have multiple triggers to make removal from these lists on
	// connection close less horrendously expensive.
	//
	// Where possible, the quit channel is used as the unique id for a client
	// since it is quite a bit more efficient than using the entire struct.
	blockNotifications := make(map[chan struct{}]*WSClient)
	txNotifications := make(map[chan struct{}]*WSClient)

out:
	for {
		select {
		case n, ok := <-m.notificationMsgs:
			if !ok {
				// queueHandler quit.
				break out
			}
			switch n := n.(type) {
			case *notificationBlockConnected:
				block := (*types.Block)(n)

				if len(blockNotifications) != 0 {
					m.notifyBlockConnected(blockNotifications, block)
				}

			case *notificationBlockDisconnected:
				block := (*types.Block)(n)
				if len(blockNotifications) != 0 {
					m.notifyBlockDisconnected(blockNotifications, block)
				}

			case *notificationTxAcceptedByMempool:
				if n.isNew && len(txNotifications) != 0 {
					m.notifyForNewTx(txNotifications, n.tx)
				}
			case *notificationRegisterBlocks:
				wsc := (*WSClient)(n)
				blockNotifications[wsc.quit] = wsc

			case *notificationUnregisterBlocks:
				wsc := (*WSClient)(n)
				delete(blockNotifications, wsc.quit)

			case *notificationRegisterClient:
				wsc := (*WSClient)(n)
				clients[wsc.quit] = wsc

			case *notificationUnregisterClient:
				wsc := (*WSClient)(n)
				delete(blockNotifications, wsc.quit)
				delete(txNotifications, wsc.quit)
				delete(clients, wsc.quit)

			case *notificationRegisterNewMempoolTxs:
				wsc := (*WSClient)(n)
				txNotifications[wsc.quit] = wsc

			case *notificationUnregisterNewMempoolTxs:
				wsc := (*WSClient)(n)
				delete(txNotifications, wsc.quit)

			default:
				log.Warnf("Unhandled notification type")
			}

		case m.numClients <- len(clients):

		case <-m.quit:
			break out
		}
	}

	for _, c := range clients {
		c.Disconnect()
	}
	m.wg.Done()
}

// NumClients returns the number of clients actively being served.
func (m *WSNotificationManager) NumClients() (n int) {
	select {
	case n = <-m.numClients:
	case <-m.quit: // Use default n (0) if server has shut down.
	}
	return
}

// RegisterBlockUpdates requests block update notifications to the passed
// websocket client.
func (m *WSNotificationManager) RegisterBlockUpdates(wsc *WSClient) {
	m.queueNotification <- (*notificationRegisterBlocks)(wsc)
}

// UnregisterBlockUpdates removes block update notifications for the passed
// websocket client.
func (m *WSNotificationManager) UnregisterBlockUpdates(wsc *WSClient) {
	m.queueNotification <- (*notificationUnregisterBlocks)(wsc)
}

// notifyBlockConnected notifies websocket clients that have registered for
// block updates when a block is connected to the main chain.
func (*WSNotificationManager) notifyBlockConnected(clients map[chan struct{}]*WSClient, block *types.Block) {

	resp, err := NewWSRequest(nil, "blockconnected", block)
	if err != nil {
		log.Errorf("Failed to build websocket response: %v", err)
		return
	}
	marshalledJSON, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Failed to marshal block connected notification: %v", err)
		return
	}

	for _, wsc := range clients {
		wsc.QueueNotification(marshalledJSON)
	}
}

// notifyBlockDisconnected notifies websocket clients that have registered for
// block updates when a block is disconnected from the main chain (due to a
// reorganize).
func (*WSNotificationManager) notifyBlockDisconnected(clients map[chan struct{}]*WSClient, block *types.Block) {
	resp, err := NewWSRequest(nil, "blockdisconnected", block)
	if err != nil {
		log.Errorf("Failed to build websocket response: %v", err)
		return
	}
	marshalledJSON, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Failed to marshal block connected notification: %v", err)
		return
	}

	for _, wsc := range clients {
		wsc.QueueNotification(marshalledJSON)
	}

}

// RegisterNewMempoolTxsUpdates requests notifications to the passed websocket
// client when new transactions are added to the memory pool.
func (m *WSNotificationManager) RegisterNewMempoolTxsUpdates(wsc *WSClient) {
	m.queueNotification <- (*notificationRegisterNewMempoolTxs)(wsc)
}

// UnregisterNewMempoolTxsUpdates removes notifications to the passed websocket
// client when new transaction are added to the memory pool.
func (m *WSNotificationManager) UnregisterNewMempoolTxsUpdates(wsc *WSClient) {
	m.queueNotification <- (*notificationUnregisterNewMempoolTxs)(wsc)
}

// notifyForNewTx notifies websocket clients that have registered for updates
// when a new transaction is added to the memory pool.
func (m *WSNotificationManager) notifyForNewTx(clients map[chan struct{}]*WSClient, tx *types.Tx) {
	resp, err := NewWSRequest(nil, "transaction", tx)
	if err != nil {
		log.Errorf("Failed to build websocket response: %v", err)
		return
	}
	marshalledJSON, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Failed to marshal tx notification: %v", err)
		return
	}

	for _, wsc := range clients {
		wsc.QueueNotification(marshalledJSON)
	}
}

// AddClient adds the passed websocket client to the notification manager.
func (m *WSNotificationManager) AddClient(wsc *WSClient) {
	m.queueNotification <- (*notificationRegisterClient)(wsc)
}

// RemoveClient removes the passed websocket client and all notifications
// registered for it.
func (m *WSNotificationManager) RemoveClient(wsc *WSClient) {
	select {
	case m.queueNotification <- (*notificationUnregisterClient)(wsc):
	case <-m.quit:
	}
}

// Start starts the goroutines required for the manager to queue and process
// websocket client notifications.
func (m *WSNotificationManager) Start() {
	m.wg.Add(2)
	go m.queueHandler()
	go m.notificationHandler()
}

// WaitForShutdown blocks until all notification manager goroutines have
// finished.
func (m *WSNotificationManager) WaitForShutdown() {
	m.wg.Wait()
}

// Shutdown shuts down the manager, stopping the notification queue and
// notification handler goroutines.
func (m *WSNotificationManager) Shutdown() {
	close(m.quit)
}

// newWsNotificationManager returns a new notification manager ready for use.
// See WSNotificationManager for more details.
func NewWsNotificationManager() *WSNotificationManager {
	return &WSNotificationManager{
		queueNotification: make(chan interface{}),
		notificationMsgs:  make(chan interface{}),
		numClients:        make(chan int),
		quit:              make(chan struct{}),
	}
}
