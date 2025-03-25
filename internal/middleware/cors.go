package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

func Cors() func(http.Handler) http.Handler {
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
	return cors.New(options).Handler
}
