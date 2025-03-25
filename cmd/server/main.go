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

	"github.com/lmittmann/tint"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/database"
	"github.com/vancomm/minesweeper-server/internal/middleware"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func main() {
	var handler slog.Handler = slog.NewJSONHandler(os.Stderr, nil)
	if config.Development() {
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := database.Connect(ctx)
	if err != nil {
		logger.Error("failed to connect and migrate db", "error", err)
	}

	cookies, err := config.NewCookies()
	if err != nil {
		logger.Error("failed to read cookies config", slog.Any("error", err))
		return
	}

	jwt, err := config.NewJWT()
	if err != nil {
		logger.Error("failed to read jwt config", slog.Any("error", err))
		return
	}

	ws, err := config.NewWebSocket()
	if err != nil {
		logger.Error("failed to read ws config", "error", err)
		return
	}

	port := config.Port()

	app := &application{
		logger:  logger,
		repo:    repository.New(db),
		ws:      ws,
		cookies: cookies,
		jwt:     jwt,
		rnd:     createRand(),
	}
	router := app.Router()
	router.Use(middleware.Cors(), middleware.Logging(logger))
	server := &http.Server{
		Addr:         port,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	errCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to listen and serve: %w", err)
		}
		close(errCh)
	}()

	logger.Info(fmt.Sprintf("minesweeper server listening at http://localhost%s", port))

	select {
	case <-ctx.Done():
		break
	case err := <-errCh:
		logger.Error("failed to start", slog.Any("error", err))
		os.Exit(1)
	}

	sCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	server.Shutdown(sCtx)
}
