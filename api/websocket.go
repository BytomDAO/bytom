package api

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/net/websocket"
)

// timeZeroVal is simply the zero value for a time.Time and is used to avoid
// creating multiple instances.
var timeZeroVal time.Time

// WebsocketHandler handles connections and requests from websocket client
func (a *API) websocketHandler(w http.ResponseWriter, r *http.Request) {
	log.WithField("remoteAddress", r.RemoteAddr).Info("New websocket client")

	client, err := websocket.NewWebsocketClient(w, r, a.notificationMgr)
	if err != nil {
		log.WithField("error", err).Error("Failed to new websocket client")
		http.Error(w, "400 Bad Request.", http.StatusBadRequest)
		return
	}

	a.notificationMgr.AddClient(client)
	client.Start()
	client.WaitForShutdown()
	a.notificationMgr.RemoveClient(client)
	log.WithField("address", r.RemoteAddr).Infoln("Disconnected websocket client")
}
