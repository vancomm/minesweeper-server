package main

import (
	"encoding/json"
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

func (app application) ServeMux() *http.ServeMux {
	base := app.basePath
	mux := http.NewServeMux()

	mux.HandleFunc("POST "+base+"/login", app.handleLogin)
	mux.HandleFunc("POST "+base+"/register", app.handleRegister)

	mux.Handle(base+"/game", http.StripPrefix(base+"/game",
		http.Handler(app.auth(app.proxy("http://game:8080")))),
	)
	mux.Handle(base+"/game/", http.StripPrefix(base+"/game",
		http.Handler(app.auth(app.proxy("http://game:8080")))),
	)

	mux.HandleFunc(base+"/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	return mux
}

func (app *application) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-Player-ID")
		claims, err := app.cookies.ParsePlayerClaims(r)
		if err != nil {
			r.Header.Add("X-Player-ID", "anon")
		} else {
			r.Header.Add("X-Player-ID", strconv.Itoa(claims.PlayerId))
		}
		next(w, r)
	}
}

func (app application) badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("your request is invalid"))
}

func (app application) unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("you are not allowed to execute this operation"))

}

func (app application) internalError(w http.ResponseWriter, msg string, args ...any) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("internal error"))
	app.logger.Error(msg, args...)
}

func (app application) badGateway(w http.ResponseWriter, msg string, args ...any) {
	w.WriteHeader(http.StatusBadGateway)
	w.Write([]byte("failed to make proxy request"))
	app.logger.Error(msg, args...)
}

func (app application) replyWith(w http.ResponseWriter, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		app.internalError(w, "failed to marshal json", err)
		return
	}
	_, err = w.Write(payload)
	if err != nil {
		app.logger.Error(
			"failed to send data",
			slog.Any("data", v),
			slog.Any("error", err),
		)
	}
}
