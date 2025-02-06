package main

import (
	"net/http"

	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleFetchHighScores(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if query.Has("seed") {
		p, err := mines.ParseSeed(query.Get("seed"))
		if err != nil {
			badRequest(w)
			return
		}
		app.repo.GetHighScoresForGameParams(
			r.Context(), repository.GetHighScoresForGameParamsParams{
				Width:     int32(p.Width),
				Height:    int32(p.Height),
				MineCount: int32(p.MineCount),
				Unique:    p.Unique,
			},
		)
	}
}
