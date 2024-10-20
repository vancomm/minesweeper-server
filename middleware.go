package main

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/rs/cors"
)

func useMiddleware(
	h http.Handler,
	mws ...func(http.Handler) http.Handler,
) http.Handler {
	for _, mw := range mws {
		h = mw(h)
	}
	return h
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	hijacked   bool
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	w.hijacked = true
	return h.Hijack()
}

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("--> %s %s", r.Method, r.URL.String())
		wrapped := &loggingResponseWriter{w, http.StatusOK, false}
		h.ServeHTTP(wrapped, r)
		message := "<-- %d %s"
		if wrapped.hijacked {
			message += " (hijacked)"
		}
		code := wrapped.statusCode
		log.Infof(message, code, http.StatusText(code))
	})
}

type requestCtxKey int

const (
	ctxPlayerClaims requestCtxKey = iota
)

func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, err := getJWTFromCookies(r); err != nil {
			// clear malformed/expired token
			log.Debug("received malformed token - clear cookies")
			clearPlayerCookies(w)
		} else {
			claims, err := tryParseJWTCookie(token)
			if err == nil {
				log.Debug("authorized as " + claims.Username)
				ctx := context.WithValue(r.Context(), ctxPlayerClaims, claims)
				r = r.WithContext(ctx)
			}
		}
		h.ServeHTTP(w, r)
	})
}

func corsMiddleware(h http.Handler) http.Handler {
	options := cors.Options{
		AllowOriginFunc: func(origin string) bool {
			return true // HACK f*ck you cors!!!
		},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}
	return cors.New(options).Handler(h)
}
