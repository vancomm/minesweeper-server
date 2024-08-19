package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vancomm/minesweeper-server/mines"
)

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
	query := h.URL.Query()
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
	if err := sendSessionJSON(w, &session); err != nil {
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
