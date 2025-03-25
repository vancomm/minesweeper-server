package main

import (
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

type application struct {
	logger  *slog.Logger
	repo    *repository.Queries
	cookies *config.Cookies
	jwt     *config.JWT
	ws      *config.WebSocket
	rnd     *rand.Rand
}

func (app application) Router() *mux.Router {
	router := mux.NewRouter()

	gameRouter := router.PathPrefix("/game/").Subrouter()
	gameRouter.Use(app.authenticate)
	gameRouter.Methods("GET").Path("/highscore").HandlerFunc(app.handleFetchHighScore)
	gameRouter.Methods("GET").Path("/{id}/connect").HandlerFunc(app.wsConnect)
	gameRouter.Methods("POST").Path("/{id}/forfeit").HandlerFunc(app.handleForfeit)
	gameRouter.Methods("POST").Path("/{id}/move").HandlerFunc(app.handleMove)
	gameRouter.Methods("GET").Path("/{id}").HandlerFunc(app.handleFetchGame)
	gameRouter.Methods("POST").HandlerFunc(app.handleNewGame)

	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/login", app.handleLogin)
	router.HandleFunc("/register", app.handleRegister)
	router.HandleFunc("/logout", app.handleLogout)

	return router
}

func (app application) getSessionId(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	return strconv.Atoi(vars["id"])
}

func (app application) getAuthenticatedPlayerId(r *http.Request) (playerId int, ok bool) {
	playerIdHeader := r.Header.Get("X-Player-ID")
	playerId, err := strconv.Atoi(playerIdHeader)
	return playerId, err == nil
}

func (app application) badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Bad request"))
}

func (app application) unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

func (app application) notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found"))
}

func (app application) internalError(w http.ResponseWriter, msg string, args ...any) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal error"))
	app.logger.Error(msg, args...)
}

func (app application) replyWithJSON(w http.ResponseWriter, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		app.internalError(w, "failed to marshal json", err)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		w.Header().Del("Content-Type")
		app.logger.Error(
			"failed to send data", slog.Any("data", v), slog.Any("error", err),
		)
	}
}
