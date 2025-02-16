package middleware

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type loggingWriter struct {
	http.ResponseWriter
	statusCode int
	hijacked   bool
}

func (w *loggingWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// loggingWriter implements [http.Hijacker]
func (w *loggingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	w.hijacked = true
	return h.Hijack()
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info(">>", slog.String("method", r.Method), slog.Any("uri", r.URL.RequestURI()))

			start := time.Now()
			wrapped := &loggingWriter{ResponseWriter: w}
			next.ServeHTTP(wrapped, r)

			logLevel := slog.LevelInfo
			if wrapped.statusCode >= 400 {
				logLevel = slog.LevelWarn
			} else if wrapped.statusCode >= 500 {
				logLevel = slog.LevelError
			}

			logger.Log(context.Background(), logLevel,
				"<<",
				slog.String("method", r.Method),
				slog.String("uri", r.URL.RequestURI()),
				slog.Int("statusCode", wrapped.statusCode),
				slog.Bool("hijacked", wrapped.hijacked),
				slog.String("remoteAddr", r.RemoteAddr),
				slog.String("xffHeader", r.Header.Get("X-Forwarded-For")),
				slog.Any("duration", time.Since(start).Milliseconds()),
			)
		})
	}
}
