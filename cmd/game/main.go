package main

import (
	"context"
	"errors"
	"fmt"
	"hash/maphash"
	"log/slog"
	"math/rand/v2"
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

func createRand() *rand.Rand {
	return rand.New(rand.NewPCG(
		new(maphash.Hash).Sum64(), new(maphash.Hash).Sum64(),
	))
}

func main() {
	var logger *slog.Logger
	if config.Development() {
		logger = slog.New(
			tint.NewHandler(os.Stderr, &tint.Options{Level: slog.LevelDebug}),
		)

	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := database.Connect(ctx)
	if err != nil {
		logger.Error("failed to connect and migrate db", "error", err)
	}

	ws, err := config.NewWebSocket()
	if err != nil {
		logger.Error("failed to read ws config", "error", err)
		return
	}

	basePath := config.BasePath()
	port := config.Port()

	app := &application{
		logger: logger,
		repo:   repository.New(db),
		ws:     ws,
		rnd:    createRand(),
	}
	server := &http.Server{
		Addr:    port,
		Handler: middleware.Logging(logger)(app.ServeMux()),
	}

	errCh := make(chan error, 1)

	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to listen and serve: %w", err)
		}
		close(errCh)
	}()

	logger.Info("game online", slog.String("port", port), slog.String("base path", basePath))

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
