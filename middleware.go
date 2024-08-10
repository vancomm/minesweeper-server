package main

import "net/http"

type Middleware func(http.Handler) http.Handler

func useMiddleware(s *http.ServeMux, mws ...Middleware) http.Handler {
	var h http.Handler = s
	for _, mw := range mws {
		h = mw(h)
	}
	return h
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("--> %s %s", r.Method, r.URL.String())
		wrapped := &loggingResponseWriter{w, http.StatusOK}
		h.ServeHTTP(wrapped, r)
		code := wrapped.statusCode
		log.Infof("<-- %d %s", code, http.StatusText(code))
	})
}
