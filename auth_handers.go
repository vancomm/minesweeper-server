package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func handleRegister(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("body must contain url-encoded username and password"))
		return
	}
	passwordBytes := []byte(password)
	if len(passwordBytes) > 72 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("password must not exceed 72 bytes"))
		return
	}
	hash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error(err)
		return
	}
	player, err := pg.CreatePlayer(context.Background(), username, hash)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("username taken"))
		return
	} else if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error("unable to insert player: ", err)
		return
	}
	token, err := createPlayerToken(player.PlayerId, player.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("unable to sign jwt token: ", err)
		return
	}
	setPlayerCookies(w, token)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("body must contain url-encoded username and password"))
		return
	}
	player, err := pg.GetPlayer(context.Background(), username)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("username unknown"))
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if err := bcrypt.CompareHashAndPassword(
		player.PasswordHash, []byte(password),
	); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token, err := createPlayerToken(player.PlayerId, player.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("unable to sign jwt token: ", err)
		return
	}
	setPlayerCookies(w, token)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	clearPlayerCookies(w)
}
