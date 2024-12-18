package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/middleware"
	"github.com/vancomm/minesweeper-server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	logger  *slog.Logger
	repo    *repository.Queries
	cookies *config.Cookies
	jwt     *config.JWT
}

func NewAuth(
	logger *slog.Logger,
	db *pgxpool.Pool,
	cookies *config.Cookies,
	jwt *config.JWT,
) *Auth {
	auth := &Auth{
		logger:  logger,
		repo:    repository.New(db),
		cookies: cookies,
		jwt:     jwt,
	}

	return auth
}

type PlayerInfo struct {
	PlayerId int64  `json:"player_id"`
	Username string `json:"username"`
}

type Status struct {
	LoggedIn bool        `json:"logged_in"`
	Player   *PlayerInfo `json:"player,omitempty"`
}

func (a Auth) Status(w http.ResponseWriter, r *http.Request) {
	var status *Status
	claims, ok := r.Context().Value(middleware.CtxPlayerClaims).(*config.PlayerClaims)
	if ok {
		status = &Status{
			LoggedIn: true,
			Player:   &PlayerInfo{claims.PlayerId, claims.Username},
		}
		a.logger.Debug("refresh cookies")
		token, err := a.jwt.Sign(claims)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			a.logger.Error("unable to tokenize checked claim", "error", err)
			return
		}
		a.cookies.Refresh(w, token)
	} else {
		status = &Status{LoggedIn: false, Player: nil}
		a.logger.Debug("could not parse cookies - clear cookies")
		a.cookies.Clear(w)
	}

	sendJSONOrLog(w, a.logger, status)
}

var (
	ErrBadAuthBody        = fmt.Errorf("request body must contain url-encoded username and password")
	ErrBadPasswordTooLong = fmt.Errorf("password too long")
	ErrUsernameTaken      = fmt.Errorf("username taken")
)

func (h Auth) Register(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		sendJSONOrLog(w, h.logger, wrapError(ErrBadAuthBody))
		return
	}

	passwordBytes := []byte(password)
	if len(passwordBytes) > 72 {
		w.WriteHeader(http.StatusBadRequest)
		sendJSONOrLog(w, h.logger, wrapError(ErrBadPasswordTooLong))
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
		sendJSONOrLog(w, h.logger, wrapError(ErrUsernameTaken))
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

	h.cookies.Refresh(w, token)
}
