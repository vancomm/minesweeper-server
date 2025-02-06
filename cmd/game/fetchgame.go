package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/handlers"
)

func (app application) handleFetchGame(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		notFound(w)
		return
	}

	session, err := app.repo.GetSession(r.Context(), sessionId)
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w)
		return
	}
	if err != nil {
		internalError(w)
		app.logger.Error("unable to fetch session from db", "error", err)
		return
	}

	playerId, ok := getAuthenticatedPlayerId(r)
	if ok && session.PlayerID != nil && *session.PlayerID != playerId {
		unauthorized(w)
		return
	}

	handlers.SendJSONOrLog(w, app.logger, session)
}
