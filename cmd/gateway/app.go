package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/handlers"
	"github.com/vancomm/minesweeper-server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type application struct {
	mount   string
	logger  *slog.Logger
	repo    *repository.Queries
	cookies *config.Cookies
	jwt     *config.JWT
}

var (
	ErrBadAuthBody        = fmt.Errorf("request body must contain url-encoded username and password")
	ErrBadPasswordTooLong = fmt.Errorf("password too long")
	ErrUsernameTaken      = fmt.Errorf("username taken")
)

func (app application) ServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+app.mount+"/login", app.handleLogin)
	mux.HandleFunc("POST "+app.mount+"/register", app.handleRegister)
	mux.HandleFunc(app.mount+"/game/", app.authenticate(
		app.proxy("http://game:8080"),
	))
	mux.HandleFunc(app.mount+"/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})
	return mux
}

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

func (app *application) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := app.cookies.ParsePlayerClaims(r)
		if err != nil {
			r.Header.Add("X-Player-ID", "anon")
		} else {
			r.Header.Add("X-Player-ID", strconv.FormatInt(claims.PlayerId, 10))
		}
		next(w, r)
	}
}

func (app *application) proxy(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL := target + strings.TrimPrefix(r.URL.RequestURI(), app.mount)
		req, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			app.logger.Error("failed to create proxy request", "error", err)
			return
		}

		req.Header = r.Header

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			app.logger.Error("failed to make proxy request", "error", err)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			app.logger.Error("failed to proxy response body", "error", err)
			return
		}
	}
}
