package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/handlers"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleMove(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	move, err := ParseGameMove(query.Get("move"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, err)
		return
	}

	p, err := parsePoint(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, err)
		return
	}

	sessionId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	session, err := app.repo.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("could not fetch session from db", "error", err)
		return
	}

	playerId, ok := getAuthenticatedPlayerId(r)
	if ok && session.PlayerID != nil && *session.PlayerID != playerId {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	game, err := mines.ParseGameStateFromBytes(session.State)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("db returned invalid game_session.state", "error", err)
		return
	}

	if !game.ValidatePoint(p.X, p.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch move {
	case Open:
		game.OpenCell(p.X, p.Y)
	case Flag:
		game.FlagCell(p.X, p.Y)
	case Chord:
		game.ChordCell(p.X, p.Y)
	}

	if game.Won || game.Dead {
		game.RevealMines()
		*session.EndedAt = time.Now().UTC()
	}

	b, err := game.Bytes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("unable to serialize game state", "error", err)
		return
	}

	err = app.repo.UpdateSession(r.Context(), repository.UpdateSessionParams{
		GameSessionID: session.GameSessionID,
		State:         b,
		Dead:          game.Dead,
		Won:           game.Won,
		EndedAt:       session.EndedAt,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("unable to update session in db", "error", err)
		return
	}

	handlers.SendJSONOrLog(w, app.logger, session)
}
