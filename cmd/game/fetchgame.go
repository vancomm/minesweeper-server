package main

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func (app application) handleFetchGame(w http.ResponseWriter, r *http.Request) {
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

	dto, err := NewGameSessionDTO(*session)
	if err != nil {
		app.internalError(w, "failed to create game session dto", slog.Any("error", err))
	}

	app.replyWith(w, dto)
}
