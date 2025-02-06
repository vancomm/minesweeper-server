package main

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/handlers"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		app.logger.Debug("could not parse request form")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, ErrBadAuthBody)
		return
	}

	player, err := app.repo.GetPlayer(r.Context(), username)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusUnauthorized)
		app.logger.Debug("username not found")
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("could not fetch player from db", "error", err)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		player.PasswordHash, []byte(password),
	)
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		w.WriteHeader(http.StatusUnauthorized)
		app.logger.Debug("wrong password")
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		app.logger.Error("bcrypt compare error", "error", err)
		return
	}

	claims := config.NewPlayerClaims(player.PlayerID, player.Username)
	token, err := app.jwt.Sign(claims)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("failed to sign player claims", "error", err)
		return
	}

	err = app.cookies.Refresh(w, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("failed to set auth cookies", "error", err)
	}

	handlers.SendMessageOrLog(w, app.logger, "ok")
}
