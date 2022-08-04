package livereload

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	JS = "new WebSocket(`ws://${location.host}/ws`).onmessage = () => location.reload()"
)

type livereload struct {
	sockets  map[*websocket.Conn]bool
	upgrader websocket.Upgrader
}

func New() livereload {
	return livereload{
		sockets: map[*websocket.Conn]bool{},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (lr livereload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := lr.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	lr.sockets[ws] = true
}

func (lr *livereload) Notify() {
	for ws := range lr.sockets {
		err := ws.WriteMessage(websocket.TextMessage, []byte("hello"))

		if err != nil {
			// Assume this means the socket has been closed
			delete(lr.sockets, ws)
			ws.Close()
		}
	}
}
