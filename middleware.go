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

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("%s %s", r.Method, r.URL.String())
		h.ServeHTTP(w, r)
	})
}
