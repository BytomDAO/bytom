package api

import (
	"net/http"
	"time"

	"github.com/bytom/net/websocket"
	log "github.com/sirupsen/logrus"
)

// timeZeroVal is simply the zero value for a time.Time and is used to avoid
// creating multiple instances.
var timeZeroVal time.Time

func (a *API) websocketHandler(w http.ResponseWriter, r *http.Request) {
	client, err := websocket.NewWebsocketClient(w, r, a.NtfnMgr)
	if err != nil {
		return
	}
	a.NtfnMgr.AddClient(client)
	client.Start()
	client.WaitForShutdown()
	a.NtfnMgr.RemoveClient(client)
	log.WithField("address", r.RemoteAddr).Infoln("Disconnected websocket client")
}
