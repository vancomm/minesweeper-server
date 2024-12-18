package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/vancomm/minesweeper-server/internal/config"
)

type CtxKey int

const (
	CtxPlayerClaims CtxKey = iota
)

type authWriter struct {
	http.ResponseWriter
	r       *http.Request
	cookies *config.Cookies
}

func (w authWriter) Write(b []byte) (int, error) {

	return w.ResponseWriter.Write(b)
}

func (w authWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func Auth(log *slog.Logger, cookies *config.Cookies) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := cookies.ParsePlayerClaims(r)
			if err != nil {
				cookies.Clear(w)
				h.ServeHTTP(w, r)
				return
			}
			wrapped := authWriter{w, r, cookies}
			ctx := context.WithValue(r.Context(), CtxPlayerClaims, claims)
			h.ServeHTTP(wrapped, r.WithContext(ctx))
		})
	}
}
