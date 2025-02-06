package main

import (
	"io"
	"log/slog"
	"net/http"
)

func (app *application) proxy(host string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest(r.Method, host+r.URL.RequestURI(), r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			app.logger.Error("failed to create proxy request", slog.Any("error", err))
			return
		}

		client := &http.Client{}
		req.Header = r.Header
		resp, err := client.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			app.logger.Error("failed to make proxy request", slog.Any("error", err))
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			app.logger.Error("failed to proxy response body", slog.Any("error", err))
			return
		}
	}
}
