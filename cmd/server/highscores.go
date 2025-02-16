package main

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleFetchHighScore(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := repository.HighscoreFilter{}

	if query.Has("seed") {
		gameParams, err := mines.ParseGameSeed(query.Get("seed"))
		if err != nil {
			app.badRequest(w)
			return
		}
		filter.GameParams = gameParams
	}

	if query.Has("username") {
		(*filter.Username) = query.Get("username")
	}

	highscores, err := app.repo.GetHighscores(r.Context(), filter)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		app.internalError(w,
			"failed to fetch highscores", slog.Any("err", err), slog.Any("filter", filter),
		)
		return
	}

	app.replyWithJSON(w, highscores)
}
