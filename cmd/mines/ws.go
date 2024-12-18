package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		log.Debug("\tws origin: ", r.Host)
		return true
	},
}

func handleConnectWs(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := pg.GetSession(context.Background(), sessionId)
	if err == pgx.ErrNoRows {
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
		log.Debug("\t> ", text)
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
		if err := pg.UpdateGameSession(
			context.Background(), session,
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatal(err)
		}
		if err := c.WriteJSON(session); err != nil {
			log.Error("write: ", err)
			break
		}
		log.Debug("\t< <session data>")
	}
}
