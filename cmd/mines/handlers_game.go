package main

import (
	"hash/maphash"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/schema"
	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/mines"
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
	if !params.ValidatePoint(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	game, err := mines.NewGame(&params, posParams.X, posParams.Y, rnd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	var session *GameSession
	if claims, ok := r.Context().Value(ctxPlayerClaims).(*PlayerClaims); ok {
		log.Debug("creating session for player ", claims.Username)
		session, err = pg.CreatePlayerGameSession(
			r.Context(), claims.PlayerId, game,
		)
		refreshPlayerCookies(w, *claims)
	} else {
		log.Debug("creating anonymous session")
		session, err = pg.CreateAnonymousGameSession(r.Context(), game)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
	}
	if _, err := sendJSON(w, session); err != nil {
		log.Error(err)
	}
}

func handleGetGame(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := pg.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if _, err := sendJSON(w, &session); err != nil {
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
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := pg.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if !session.State.ValidatePoint(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.State.OpenCell(posParams.X, posParams.Y)
	if session.State.Won || session.State.Dead {
		session.State.RevealMines()
		session.EndedAt = time.Now().UTC()
	}
	if err := pg.UpdateGameSession(r.Context(), session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if _, err := sendJSON(w, &session); err != nil {
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
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := pg.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if !session.State.ValidatePoint(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.State.FlagCell(posParams.X, posParams.Y)
	if err := pg.UpdateGameSession(r.Context(), session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if _, err := sendJSON(w, &session); err != nil {
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
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := pg.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if !session.State.ValidatePoint(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.State.ChordCell(posParams.X, posParams.Y)
	if session.State.Won || session.State.Dead {
		session.State.RevealMines()
		session.EndedAt = time.Now().UTC()
	}
	if err := pg.UpdateGameSession(r.Context(), session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if _, err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

func handleReveal(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := pg.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	session.State.RevealAll()
	if session.EndedAt.IsZero() {
		session.EndedAt = time.Now().UTC()
	}
	if err := pg.UpdateGameSession(r.Context(), session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if _, err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}

// // Accepts newline-separated commands transferred via body of following syntax:
// //
// //	o x y // open a square at x:y
// //	c x y // chord a square at x:y
// //	f x y // flag a square at x:y
// //
// // Commands are interpreted in the order they are listed. If any command results
// // in a game over, interpretation stops and game state is returned immediately.
// // If any command is malformed, all changes to game state will be dropped and
// // response will have a status of [http.StatusBadRequest] and a payload with
// // command's line number and an error message.
func handleBatch(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := pg.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
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
			if _, err := sendJSON(w, payload); err != nil {
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
	if err := pg.UpdateGameSession(r.Context(), session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
	if _, err := sendJSON(w, &session); err != nil {
		log.Error(err)
	}
}
