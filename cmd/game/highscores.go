package main

import (
	"log/slog"
	"net/http"

	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleFetchHighScores(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := repository.HighscoreFilter{}

	if query.Has("seed") {
		gameParams, err := mines.ParseSeed(query.Get("seed"))
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
	if err != nil {
		app.internalError(w, "failed to fetch highscores",
			slog.Any("err", err), slog.Any("filter", filter))
		return
	}

	app.replyWith(w, highscores)
}
