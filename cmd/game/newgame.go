package main

import (
	"log/slog"
	"net/http"

	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleNewGame(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	dto, err := decodeNewGame(query)
	if err != nil {
		app.badRequest(w)
		return
	}

	p, err := decodePoint(query)
	if err != nil {
		app.badRequest(w)
		return
	}

	gameParams := mines.GameParams(dto)
	if !gameParams.PointInBounds(p.X, p.Y) {
		app.badRequest(w)
		return
	}

	game, err := mines.NewGame(&gameParams, p.X, p.Y, app.rnd)
	if err != nil {
		app.internalError(w, "unable to generate a new game", slog.Any("error", err))
		return
	}

	var sessionParams repository.CreateGameSessionParams
	if playerId, ok := app.getAuthenticatedPlayerId(r); ok {
		(*sessionParams.PlayerId) = int(playerId)
	}

	session, err := app.repo.CreateGameSession(r.Context(), game, sessionParams)
	if err != nil {
		app.internalError(w, "failed to create game session", slog.Any("error", err))
		return
	}

	sessionDTO, err := NewGameSessionDTO(*session)
	if err != nil {
		app.internalError(w, "failed to create game session dto", slog.Any("error", err))
		return
	}

	app.replyWith(w, sessionDTO)
}
