package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.badRequest(w)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	player, err := app.repo.FetchPlayer(r.Context(), username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			app.unauthorized(w)
		} else {
			app.internalError(w, "could not fetch player from db", "error", err)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword(player.PasswordHash, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			app.unauthorized(w)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			app.logger.Error("bcrypt compare error", "error", err)
		}
		return
	}

	claims := config.NewPlayerClaims(player.PlayerId, player.Username)
	token, err := app.jwt.Sign(claims)
	if err != nil {
		app.internalError(w, "failed to sign player claims", "error", err)
		return
	}

	err = app.cookies.Refresh(w, token, time.Now().Add(app.jwt.TokenLifetime))
	if err != nil {
		app.internalError(w, "failed to set auth cookies", "error", err)
		return
	}

	app.replyWithJSON(w, "ok")
}
