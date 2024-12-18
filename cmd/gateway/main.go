package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/database"
	"github.com/vancomm/minesweeper-server/internal/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cookies, err := config.NewCookies()
	if err != nil {
		logger.Error("failed to read cookies config", "error", err)
		return
	}

	jwt, err := config.NewJWT()
	if err != nil {
		logger.Error("failed to read jwt config", "error", err)
		return
	}

	db, err := database.Connect(ctx)
	if err != nil {
		logger.Error("failed to connect to db", "error", err)
		return
	}

	auth := NewAuthHandler(logger, db, cookies, jwt)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", auth.handleLogin)
	mux.HandleFunc("POST /register", auth.handleRegister)
	mux.HandleFunc("/games/", auth.authenticate(
		auth.proxy("http://game:8080"),
	))

	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.Logging(logger)(mux),
	}

	errCh := make(chan error, 1)

	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to listen and serve: %w", err)
		}
		close(errCh)
	}()

	logger.Info("gateway listening", slog.String("addr", ":8080"))

	select {
	case <-ctx.Done():
		break
	case err := <-errCh:
		logger.Error("failed to start", "error", err)
		os.Exit(1)
	}

	sCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	server.Shutdown(sCtx)
}
