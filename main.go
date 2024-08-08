package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"math/rand/v2"
	"net/http"

	_ "github.com/mattn/go-sqlite3"

	"github.com/google/uuid"
	"github.com/gorilla/schema"
	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
)

var (
	log = logrus.New()
	r   = rand.New(rand.NewPCG(1, 2))
	dec = schema.NewDecoder()
	kvs *Store
)

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

type OpenParams struct {
	X int `schema:"x,required"`
	Y int `schema:"y,required"`
}

type GameStateResponse struct {
	SessionId string         `json:"session_id"`
	Grid      mines.GridInfo `json:"grid"`
	Dead      bool           `json:"dead"`
	Won       bool           `json:"won"`
}

type LocalGameState mines.GameState

func (game LocalGameState) ToResponse(sessionId string) GameStateResponse {
	return GameStateResponse{
		SessionId: sessionId,
		Grid:      game.PlayerGrid,
		Dead:      game.Dead,
		Won:       game.Won,
	}
}

func handleNewGame(w http.ResponseWriter, h *http.Request) {
	var (
		gameParams NewGameParams
		openParams OpenParams
	)

	query := h.URL.Query()

	if err := dec.Decode(&gameParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := dec.Decode(&openParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.WithFields(logrus.Fields{
		"gameParams":  gameParams,
		"clickParams": openParams,
	}).Info("new game request")

	game, err := mines.New(mines.GameParams(gameParams), openParams.X, openParams.Y, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			log.Error(err)
		}
		return
	}

	u := [16]byte(uuid.New())
	sessionId := base64.RawURLEncoding.EncodeToString(u[:])

	err = kvs.Set(sessionId, game)
	if err != nil {
		log.Fatal(err)
	}

	res := LocalGameState(*game).ToResponse(sessionId)

	resJson, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resJson)
	if err != nil {
		log.Error(err)
	}
}

func handleOpen(w http.ResponseWriter, r *http.Request) {
	var openParams OpenParams
	query := r.URL.Query()
	if err := dec.Decode(&openParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var game mines.GameState
	sessionId := r.PathValue("id")
	if err := kvs.Get(sessionId, &game); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	game.OpenSquare(openParams.X, openParams.Y)

	res := LocalGameState(game).ToResponse(sessionId)

	resJson, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resJson)
	if err != nil {
		log.Error(err)
	}

	if err = kvs.Set(sessionId, game); err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.Info("starting up")
	defer log.Info("shutting down")

	db, err := sql.Open("sqlite3", "./sessions.sqlite")
	if err != nil {
		log.Fatalf("failed to connect sqlite db: %v", err)
	}
	defer db.Close()

	kvs, err = NewStore(db, "sessions")
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /status",
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
	mux.HandleFunc("GET /newgame", handleNewGame)
	mux.HandleFunc("GET /{id}/open", handleOpen)

	s := useMiddleware(mux, loggingMiddleware)

	addr := "localhost:8000"
	log.Infof("ready to serve @ %s", addr)

	log.Fatal(http.ListenAndServe(addr, s))
}
