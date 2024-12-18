package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/handlers"
	"github.com/vancomm/minesweeper-server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	logger  *slog.Logger
	repo    *repository.Queries
	cookies *config.Cookies
	jwt     *config.JWT
}

func NewAuthHandler(
	logger *slog.Logger,
	db *pgxpool.Pool,
	cookies *config.Cookies,
	jwt *config.JWT,
) *AuthHandler {
	auth := &AuthHandler{
		logger:  logger,
		repo:    repository.New(db),
		cookies: cookies,
		jwt:     jwt,
	}
	return auth
}

var (
	ErrBadAuthBody        = fmt.Errorf("request body must contain url-encoded username and password")
	ErrBadPasswordTooLong = fmt.Errorf("password too long")
	ErrUsernameTaken      = fmt.Errorf("username taken")
)

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Debug("could not parse request form")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, h.logger, ErrBadAuthBody)
		return
	}

	player, err := h.repo.GetPlayer(r.Context(), username)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusUnauthorized)
		h.logger.Debug("username not found")
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("could not fetch player from db", "error", err)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		player.PasswordHash, []byte(password),
	)
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		w.WriteHeader(http.StatusUnauthorized)
		h.logger.Debug("wrong password")
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.logger.Error("bcrypt compare error", "error", err)
		return
	}

	claims := config.NewPlayerClaims(player.PlayerID, player.Username)
	token, err := h.jwt.Sign(claims)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("failed to sign player claims", "error", err)
		return
	}

	err = h.cookies.Refresh(w, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("failed to set auth cookies", "error", err)
	}

	handlers.SendMessageOrLog(w, h.logger, "ok")
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, h.logger, ErrBadAuthBody)
		return
	}

	passwordBytes := []byte(password)
	if len(passwordBytes) > 72 {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, h.logger, ErrBadPasswordTooLong)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("unable to hash password", "password", password, "error", err)
		return
	}

	player, err := h.repo.CreatePlayer(r.Context(), repository.CreatePlayerParams{
		Username:     username,
		PasswordHash: hash,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
		w.WriteHeader(http.StatusConflict)
		handlers.SendErrorOrLog(w, h.logger, ErrUsernameTaken)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("unable to insert player", "error", err)
		return
	}

	token, err := h.jwt.Sign(
		config.NewPlayerClaims(player.PlayerID, player.Username),
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("unable to create a jwt token", "error", err)
	}

	err = h.cookies.Refresh(w, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("failed to set auth cookies", "error", err)
	}

	handlers.SendMessageOrLog(w, h.logger, "ok")
}

func (h *AuthHandler) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := h.cookies.ParsePlayerClaims(r)
		if err != nil {
			r.Header.Add("X-Player-ID", "anon")
		} else {
			r.Header.Add("X-Player-ID", strconv.FormatInt(claims.PlayerId, 10))
		}
		next(w, r)
	}
}

func (h *AuthHandler) proxy(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL := target + r.URL.RequestURI()
		req, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			h.logger.Error("failed to create proxy request", "error", err)
			return
		}

		req.Header = r.Header

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			h.logger.Error("failed to make proxy request", "error", err)
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
			h.logger.Error("failed to proxy response body", "error", err)
			return
		}
	}
}
