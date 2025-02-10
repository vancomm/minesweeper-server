package middleware

import (
	"bufio"
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

func (w *loggingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	w.hijacked = true
	return h.Hijack()
}

func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info(r.Method + " " + r.URL.RequestURI())
			start := time.Now()

			wrapped := &loggingWriter{ResponseWriter: w}

			next.ServeHTTP(wrapped, r)

			logger.Info(
				"handled request",
				slog.Int("statusCode", wrapped.statusCode),
				slog.Bool("hijacked", wrapped.hijacked),
				slog.String("remoteAddr", r.RemoteAddr),
				slog.String("xffHeader", r.Header.Get("X-Forwarded-For")),
				slog.String("method", r.Method),
				slog.String("uri", r.URL.RequestURI()),
				slog.Any("duration (ms)", int64(time.Since(start)/time.Millisecond)),
			)
		})
	}
}
