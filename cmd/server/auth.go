package main

import (
	"net/http"
	"strconv"
)

func (app application) authenticate(resource http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-Player-ID")
		claims, err := app.jwt.ParsePlayerClaims(r)
		if err != nil {
			r.Header.Add("X-Player-ID", "anon")
		} else {
			r.Header.Add("X-Player-ID", strconv.Itoa(claims.PlayerId))
		}
		resource.ServeHTTP(w, r)
	})
}
