package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

func handleConnectWs(w http.ResponseWriter, r *http.Request) {
	sessionId := r.PathValue("id")
	var session GameSession
	if err := kvs.Get(sessionId, &session); err == ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("upgrade: ", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Warn("read: ", err)
			}
			break
		}
		if mt != websocket.TextMessage {
			break
		}
		text := strings.TrimSpace(string(message))
		for _, c := range byPiece(text, "\n") {
			if err := executeCommand(&session.State, c); err != nil {
				log.Error("command: ", err)
				return
			}
			if session.State.Won || session.State.Dead {
				session.State.RevealMines()
				session.EndedAt = time.Now().UTC()
				break
			}
		}
		if err := kvs.Set(sessionId, session); err != nil {
			log.Fatal(err)
		}
		if err := c.WriteJSON(session); err != nil {
			log.Error("write: ", err)
			break
		}
	}
}
