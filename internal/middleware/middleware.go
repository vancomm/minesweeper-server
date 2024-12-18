package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

func Wrap(h http.Handler, mws ...Middleware) http.Handler {
	for _, mw := range mws {
		h = mw(h)
	}
	return h
}
