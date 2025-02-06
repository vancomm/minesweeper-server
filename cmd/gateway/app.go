package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

type application struct {
	basePath string
	logger   *slog.Logger
	repo     *repository.Queries
	cookies  *config.Cookies
	jwt      *config.JWT
}

var (
	ErrBadAuthBody        = fmt.Errorf("request body must contain url-encoded username and password")
	ErrBadPasswordTooLong = fmt.Errorf("password too long")
	ErrUsernameTaken      = fmt.Errorf("username taken")
)

func (app *application) auth(next http.HandlerFunc) http.HandlerFunc {
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

func (app application) ServeMux() *http.ServeMux {
	base := app.basePath
	mux := http.NewServeMux()

	mux.HandleFunc("POST "+base+"/login", app.handleLogin)
	mux.HandleFunc("POST "+base+"/register", app.handleRegister)

	mux.Handle(base+"/game/", http.StripPrefix(base+"/game",
		http.Handler(app.auth(app.proxy("http://game:8080")))),
	)

	mux.HandleFunc(base+"/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	return mux
}
