package middleware

import (
	"net/http"
	"slices"
)

type Middleware func(http.Handler) http.Handler

func Wrap(h http.Handler, mws ...Middleware) http.Handler {
	for _, mw := range mws {
		h = mw(h)
	}
	return h
}

func Chain(mws ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for _, mw := range slices.Backward(mws) {
			next = mw(next)
		}
		return next
	}
}
