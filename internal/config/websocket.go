package config

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocket struct {
	Upgrader websocket.Upgrader
}

func NewWebSocket() (*WebSocket, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ws := &WebSocket{
		Upgrader: upgrader,
	}

	return ws, nil
}
