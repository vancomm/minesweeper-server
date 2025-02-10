package main

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleForfeit(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		app.notFound(w)
		return
	}

	session, err := app.repo.FetchGameSession(r.Context(), sessionId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			app.notFound(w)
		} else {
			app.internalError(w, "unable to fetch session from db", err)
		}
		return
	}

	playerId, ok := app.getAuthenticatedPlayerId(r)
	if ok && session.PlayerId != nil && *session.PlayerId != playerId {
		app.unauthorized(w)
		return
	}

	game, err := mines.DecodeGameState(session.State)
	if err != nil {
		app.badRequest(w)
		return
	}

	game.RevealPlayerGrid()
	if session.EndedAt.Time.IsZero() {
		session.EndedAt.Time = time.Now().UTC()
	}

	b, err := game.Bytes()
	if err != nil {
		app.internalError(w, "unable to serialize game state", slog.Any("error", err))
		return
	}

	err = app.repo.UpdateGameSession(
		r.Context(),
		session.GameSessionId,
		repository.UpdateGameSessionParams{
			State:   &b,
			Dead:    &game.Dead,
			Won:     &game.Won,
			EndedAt: &session.EndedAt.Time,
		},
	)
	if err != nil {
		app.internalError(w, "unable to update session in db", "error", err)
		return
	}

	dto, err := NewGameSessionDTO(*session)
	if err != nil {
		app.internalError(w, "failed to create game session dto", slog.Any("error", err))
		return
	}

	app.replyWith(w, dto)
}
