package main

import (
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

type application struct {
	logger *slog.Logger
	repo   *repository.Queries
	ws     *config.WebSocket
	rnd    *rand.Rand
}

func getAuthenticatedPlayerId(r *http.Request) (playerId int64, ok bool) {
	playerIdStr := r.Header.Get("X-Player-ID")
	playerId, err := strconv.ParseInt(playerIdStr, 10, 64)
	return playerId, err == nil
}

func notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found :("))
}

func internalError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("internal error"))
}

func unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("you are not allowed to execute this operation"))
}

func badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("your request is invalid"))
}

func (app application) ServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", app.handleNewGame)
	mux.HandleFunc("GET /{id}", app.handleFetchGame)
	mux.HandleFunc("POST /{id}/move", app.handleMove)
	mux.HandleFunc("POST /{id}/forfeit", app.handleForfeit)
	mux.HandleFunc("CONNECT /{id}/connect", app.wsConnect)
	mux.HandleFunc("GET /highscores", app.handleFetchHighScores)
	return mux
}
