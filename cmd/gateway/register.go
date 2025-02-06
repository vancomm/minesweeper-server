package main

import (
	"errors"
	"net/http"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/handlers"
	"github.com/vancomm/minesweeper-server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) handleRegister(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, ErrBadAuthBody)
		return
	}

	passwordBytes := []byte(password)
	if len(passwordBytes) > 72 {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, ErrBadPasswordTooLong)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("unable to hash password", "password", password, "error", err)
		return
	}

	player, err := app.repo.CreatePlayer(r.Context(), repository.CreatePlayerParams{
		Username:     username,
		PasswordHash: hash,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
		w.WriteHeader(http.StatusConflict)
		handlers.SendErrorOrLog(w, app.logger, ErrUsernameTaken)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("unable to insert player", "error", err)
		return
	}

	token, err := app.jwt.Sign(
		config.NewPlayerClaims(player.PlayerID, player.Username),
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("unable to create a jwt token", "error", err)
	}

	err = app.cookies.Refresh(w, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("failed to set auth cookies", "error", err)
	}

	handlers.SendMessageOrLog(w, app.logger, "ok")
}
