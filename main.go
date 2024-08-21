package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"hash/maphash"
	"math/rand/v2"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"

	"github.com/gorilla/schema"
	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
)

var port int

func init() {
	const (
		defaultPort = 8000
		usage       = "listening port"
	)
	flag.IntVar(&port, "port", defaultPort, usage)
	flag.IntVar(&port, "p", defaultPort, usage+" (shorthand)")
}

var (
	log = logrus.New()
	dec = schema.NewDecoder()
	r   = rand.New(rand.NewPCG(
		new(maphash.Hash).Sum64(),
		new(maphash.Hash).Sum64(),
	))
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

type PosParams struct {
	X int `schema:"x,required"`
	Y int `schema:"y,required"`
}

type GameSession struct {
	SessionId string
	State     mines.GameState
	StartedAt time.Time
	EndedAt   time.Time
}

type GameSessionJSON struct {
	SessionId string         `json:"session_id"`
	Grid      mines.GridInfo `json:"grid"`
	Width     int            `json:"width"`
	Height    int            `json:"height"`
	MineCount int            `json:"mine_count"`
	Unique    bool           `json:"unique"`
	Dead      bool           `json:"dead"`
	Won       bool           `json:"won"`
	StartedAt int64          `json:"started_at"`
	EndedAt   *int64         `json:"ended_at,omitempty"`
}

func (s GameSession) MarshalJSON() ([]byte, error) {
	var endedAt *int64
	if !s.EndedAt.IsZero() {
		e := s.EndedAt.Unix()
		endedAt = &e
	}
	return json.Marshal(GameSessionJSON{
		SessionId: s.SessionId,
		Grid:      s.State.PlayerGrid,
		Width:     s.State.Width,
		Height:    s.State.Height,
		MineCount: s.State.MineCount,
		Unique:    s.State.Unique,
		Dead:      s.State.Dead,
		Won:       s.State.Won,
		StartedAt: s.StartedAt.Unix(),
		EndedAt:   endedAt,
	})
}

func sendJSON(w http.ResponseWriter, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	_, err = w.Write(payload)
	return err
}

func main() {
	flag.Parse()
	log.Info("starting up")

	db, err := sql.Open("sqlite3", "./data/sessions.sqlite")
	if err != nil {
		log.Fatalf("failed to connect sqlite db: %v", err)
	}
	defer db.Close()

	kvs, err = NewStore(db, "sessions")
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/status", handleStatus)
	mux.HandleFunc("POST /v1/game", handleNewGame)
	mux.HandleFunc("GET /v1/game/{id}", handleGetState)
	mux.HandleFunc("POST /v1/game/{id}/open", handleOpen)
	mux.HandleFunc("POST /v1/game/{id}/flag", handleFlag)
	mux.HandleFunc("POST /v1/game/{id}/chord", handleChord)
	mux.HandleFunc("POST /v1/game/{id}/reveal", handleReveal)
	mux.HandleFunc("POST /v1/game/{id}/batch", handleBatch)

	mux.HandleFunc("/v1/game/{id}/connect", handleConnectWs)

	h := useMiddleware(mux,
		loggingMiddleware,
		cors.Default().Handler,
	)

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Infof("ready to serve @ %s", addr)

	log.Fatal(http.ListenAndServe(addr, h))
}
