package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
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

func sendJSON(w http.ResponseWriter, v any) error {
	payload, err := json.Marshal(v)
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

func executeBatchCommand(s *mines.GameState, c string) (err error) {
	parts := strings.Split(c, " ")
	if len(parts) != 3 {
		return errors.New("commands must have three parts")
	}
	var x, y int
	if x, err = strconv.Atoi(parts[1]); err != nil {
		return errors.New("second command argument must be an int")
	}
	if y, err = strconv.Atoi(parts[2]); err != nil {
		return errors.New("third command argument must be an int")
	}
	if !s.ValidateSquare(x, y) {
		return errors.New("invalid square coordinates")
	}
	switch parts[0] {
	case "o":
		s.OpenSquare(x, y)
	case "c":
		s.ChordSquare(x, y)
	case "f":
		s.FlagSquare(x, y)
	default:
		return errors.New("invalid command")
	}
	return
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
		if err := executeBatchCommand(&session.State, c); err != nil {
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
