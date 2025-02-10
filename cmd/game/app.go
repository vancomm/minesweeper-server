package main

import (
	"encoding/json"
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

func (app application) ServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", app.handleNewGame)
	mux.HandleFunc("GET /{id}", app.handleFetchGame)
	mux.HandleFunc("POST /{id}/move", app.handleMove)
	mux.HandleFunc("POST /{id}/forfeit", app.handleForfeit)
	mux.HandleFunc("GET /highscores", app.handleFetchHighScores)
	mux.HandleFunc("GET /{id}/connect", app.wsConnect)
	return mux
}

func (app application) getAuthenticatedPlayerId(r *http.Request) (playerId int, ok bool) {
	playerIdStr := r.Header.Get("X-Player-ID")
	playerId, err := strconv.Atoi(playerIdStr)
	return playerId, err == nil
}

func (app application) badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("your request is invalid"))
}

func (app application) unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("you are not allowed to execute this operation"))

}
func (app application) notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found :("))
}

func (app application) internalError(w http.ResponseWriter, msg string, args ...any) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("internal error"))
	app.logger.Error(msg, args...)
}

func (app application) replyWith(w http.ResponseWriter, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		app.internalError(w, "failed to marshal json", err)
		return
	}
	_, err = w.Write(payload)
	if err != nil {
		app.logger.Error(
			"failed to send data",
			slog.Any("data", v),
			slog.Any("error", err),
		)
	}
}
