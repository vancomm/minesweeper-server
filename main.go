package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"hash/maphash"
	"math/rand/v2"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"

	"github.com/google/uuid"
	"github.com/gorilla/schema"
	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
)

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

func NewGameSession(state mines.GameState) *GameSession {
	u := [16]byte(uuid.New())
	sessionId := base64.RawURLEncoding.EncodeToString(u[:])
	return &GameSession{
		SessionId: sessionId,
		State:     state,
		StartedAt: time.Now().UTC(),
	}
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

func sendSessionJSON(w http.ResponseWriter, session *GameSession) error {
	payload, err := json.Marshal(session)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	_, err = w.Write(payload)
	return err
}

func handleNewGame(w http.ResponseWriter, h *http.Request) {
	var (
		gameParams NewGameParams
		posParams  PosParams
	)
	query := h.URL.Query()
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
	game, err := mines.New(params, posParams.X, posParams.Y, r)
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
	if err := sendSessionJSON(w, session); err != nil {
		log.Error(err)
	}
}

func handleGetState(w http.ResponseWriter, r *http.Request) {
	var session GameSession
	sessionId := r.PathValue("id")
	if err := kvs.Get(sessionId, &session); err == ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if err := sendSessionJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleOpen(w http.ResponseWriter, r *http.Request) {
	var posParams PosParams
	query := r.URL.Query()
	if err := dec.Decode(&posParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var session GameSession
	sessionId := r.PathValue("id")
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
		session.EndedAt = time.Now().UTC()
	}
	if err := kvs.Set(sessionId, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendSessionJSON(w, &session); err != nil {
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
	var session GameSession
	sessionId := r.PathValue("id")
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
	if err := sendSessionJSON(w, &session); err != nil {
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
	var session GameSession
	sessionId := r.PathValue("id")
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
		session.EndedAt = time.Now().UTC()
	}
	if err := kvs.Set(sessionId, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if err := sendSessionJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("\"ok\""))
}

func main() {
	log.Info("starting up")

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

	mux.HandleFunc("GET /v1/status", handleStatus)
	mux.HandleFunc("POST /v1/game", handleNewGame)
	mux.HandleFunc("GET /v1/game/{id}", handleGetState)
	mux.HandleFunc("POST /v1/game/{id}/open", handleOpen)
	mux.HandleFunc("POST /v1/game/{id}/flag", handleFlag)
	mux.HandleFunc("POST /v1/game/{id}/chord", handleChord)

	h := useMiddleware(mux,
		loggingMiddleware,
		cors.Default().Handler,
	)

	addr := "localhost:8000"
	log.Infof("ready to serve @ %s", addr)

	log.Fatal(http.ListenAndServe(addr, h))
}
