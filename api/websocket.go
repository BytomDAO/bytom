package api

import (
	"net/http"
	"time"

	ws "github.com/bytom/net/websocket"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// timeZeroVal is simply the zero value for a time.Time and is used to avoid
// creating multiple instances.
var timeZeroVal time.Time

func (a *API) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Attempt to upgrade the connection to a websocket connection
	// using the default size for read/write buffers.
	ws, err := websocket.Upgrade(w, r, nil, 0, 0)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Printf("Unexpected websocket error: %v", err)
		}
		http.Error(w, "400 Bad Request.", http.StatusBadRequest)
		return
	}
	a.newWebsocketClient(ws, r.RemoteAddr)
}

func (a *API) newWebsocketClient(conn *websocket.Conn, remoteAddr string) {

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
