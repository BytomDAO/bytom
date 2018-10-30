package api

import (
	"time"

	ws "github.com/bytom/net/websocket"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/types"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// timeZeroVal is simply the zero value for a time.Time and is used to avoid
// creating multiple instances.
var timeZeroVal time.Time

func (a *API) handleBlockchainNotification(notification *protocol.Notification) {
	switch notification.Type {
	case protocol.NTBlockAccepted:

	case protocol.NTBlockConnected:
		block, ok := notification.Data.(*types.Block)
		if !ok {
			log.Errorf("Chain connected notification is not a block.")
			break
		}

		// Notify registered websocket clients of incoming block.
		a.NtfnMgr.NotifyBlockConnected(block)
	case protocol.NTBlockDisconnected:
		block, ok := notification.Data.(*types.Block)
		if !ok {
			log.Errorf("Chain disconnected notification is not a block.")
			break
		}

		// Notify registered websocket clients.
		a.NtfnMgr.NotifyBlockDisconnected(block)
	}
}

func (a *API) buildWebsocketHandler(conn *websocket.Conn, remoteAddr string) {

	// Clear the read deadline that was set before the websocket hijacked
	// the connection.
	conn.SetReadDeadline(timeZeroVal)

	// Limit max number of websocket clients.
	log.WithField("New websocket client", remoteAddr).Info("WebSocket listen")

	if a.NtfnMgr.NumClients()+1 > a.maxWebsockets {
		log.Infof("Max websocket clients exceeded [%d] - "+
			"disconnecting client %s", a.maxWebsockets, remoteAddr)
		conn.Close()
		return
	}
	client, err := ws.NewWebsocketClient(conn, remoteAddr, a.NtfnMgr, a.maxConcurrentReqs)
	//client, err := a.newWebsocketClient(conn, remoteAddr)
	if err != nil {
		log.Errorf("Failed to serve client %s: %v", remoteAddr, err)
		conn.Close()
		return
	}

	a.NtfnMgr.AddClient(client)
	client.Start()
	client.WaitForShutdown()
	a.NtfnMgr.RemoveClient(client)
	log.Infof("Disconnected websocket client %s", remoteAddr)
}
