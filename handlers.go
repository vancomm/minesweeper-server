package main

import (
	"encoding/base64"
	"hash/maphash"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/schema"
	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
)

var (
	dec = schema.NewDecoder()
	rnd = rand.New(rand.NewPCG(
		new(maphash.Hash).Sum64(),
		new(maphash.Hash).Sum64(),
	))
)

func init() {
	dec.IgnoreUnknownKeys(true)
}

func init() {
	log.SetLevel(logrus.DebugLevel)
	mines.Log.SetLevel(logrus.DebugLevel)

	dec.IgnoreUnknownKeys(true)
}

type NewGameParams struct {
	Width     int  `schema:"width,required"`
	Height    int  `schema:"height,required"`
	MineCount int  `schema:"mine_count,required"`
	Unique    bool `schema:"unique,required"`
}

type PosParams struct {
	X int `schema:"x,required"`
	Y int `schema:"y,required"`
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("\"ok\""))
}

func NewGameSession(state mines.GameState) *GameSession {
	u := [16]byte(uuid.New())
	sessionId := base64.RawURLEncoding.EncodeToString(u[:])
	return &GameSession{
		SessionId: sessionId,
		State:     state,
		StartedAt: time.Now().UTC(),
	}
}

func handleNewGame(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var (
		gameParams NewGameParams
		posParams  PosParams
	)
	if err := dec.Decode(&gameParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := dec.Decode(&posParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	params := mines.GameParams(gameParams)
	if !params.ValidateSquare(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	game, err := mines.New(params, posParams.X, posParams.Y, rnd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	session := NewGameSession(*game)
	if err = kvs.Set(session.SessionId, *session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendJSON(w, session); err != nil {
		log.Error(err)
	}
}

func handleGetState(w http.ResponseWriter, r *http.Request) {
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
	if err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleOpen(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var posParams PosParams
	if err := dec.Decode(&posParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
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
	if !session.State.ValidateSquare(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.State.OpenSquare(posParams.X, posParams.Y)
	if session.State.Won || session.State.Dead {
		session.State.RevealMines()
		session.EndedAt = time.Now().UTC()
	}
	if err := kvs.Set(sessionId, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleFlag(w http.ResponseWriter, r *http.Request) {
	var posParams PosParams
	query := r.URL.Query()
	if err := dec.Decode(&posParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionId := r.PathValue("id")
	var session GameSession
	if err := kvs.Get(sessionId, &session); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !session.State.ValidateSquare(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.State.FlagSquare(posParams.X, posParams.Y)
	if err := kvs.Set(sessionId, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleChord(w http.ResponseWriter, r *http.Request) {
	var posParams PosParams
	query := r.URL.Query()
	if err := dec.Decode(&posParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionId := r.PathValue("id")
	var session GameSession
	if err := kvs.Get(sessionId, &session); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !session.State.ValidateSquare(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.State.ChordSquare(posParams.X, posParams.Y)
	if session.State.Won || session.State.Dead {
		session.State.RevealMines()
		session.EndedAt = time.Now().UTC()
	}
	if err := kvs.Set(sessionId, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleReveal(w http.ResponseWriter, r *http.Request) {
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
	session.State.RevealAll()
	if session.State.Won || session.State.Dead {
		session.State.RevealMines()
		session.EndedAt = time.Now().UTC()
	}
	if err := kvs.Set(sessionId, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

// Accepts newline-separated commands transferred via body of following syntax:
//
//	o x y // open a square at x:y
//	c x y // chord a square at x:y
//	f x y // flag a square at x:y
//
// Commands are interpreted in the order they are listed. If any command results
// in a game over, interpretation stops and game state is returned immediately.
// If any command is malformed, all changes to game state will be dropped and
// response will have a status of [http.StatusBadRequest] and a payload with
// command's line number and an error message.
func handleBatch(w http.ResponseWriter, r *http.Request) {
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	lines := strings.TrimSpace(string(body))
	for i, c := range byPiece(lines, "\n") {
		if err := executeCommand(&session.State, c); err != nil {
			payload := struct {
				loc     int
				message string
			}{i, err.Error()}
			w.WriteHeader(http.StatusBadRequest)
			if err := sendJSON(w, payload); err != nil {
				log.Error(err)
			}
			return
		}
		if session.State.Won || session.State.Dead {
			session.State.RevealMines()
			session.EndedAt = time.Now().UTC()
			break
		}
	}
	if err := kvs.Set(sessionId, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleRecords(w http.ResponseWriter, r *http.Request) {
	records, err := compileGameRecords()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sendJSON(w, records); err != nil {
		log.Error(err)
	}
}
