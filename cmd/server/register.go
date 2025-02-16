package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) handleRegister(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.badRequest(w)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		app.badRequest(w)
		return
	}

	passwordBytes := []byte(password)
	if len(passwordBytes) > 72 {
		app.badRequest(w)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)
	if err != nil {
		app.internalError(w, "unable to hash password", "password", password, "error", err)
		return
	}

	player, err := app.repo.CreatePlayer_(
		r.Context(), repository.CreatePlayerParams{Username: username, PasswordHash: hash},
	)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			w.WriteHeader(http.StatusConflict)
			app.replyWithJSON(w, map[string]string{"error": "username taken"})
			return
		}
		app.internalError(w, "unable to insert player", "error", err)
		return
	}

	token, err := app.jwt.Sign(
		config.NewPlayerClaims(player.PlayerId, player.Username),
	)
	if err != nil {
		app.internalError(w, "unable to create a jwt token", "error", err)
		return
	}

	err = app.cookies.Refresh(w, token, time.Now().Add(app.jwt.TokenLifetime))
	if err != nil {
		app.internalError(w, "failed to set auth cookies", "error", err)
		return
	}

	app.replyWithJSON(w, "ok")
}
