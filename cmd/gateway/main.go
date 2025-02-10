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

	db, err := database.Connect(ctx)
	if err != nil {
		logger.Error("failed to connect and migrate db", "error", err)
	}

	port := config.Port()
	basePath := config.BasePath()
	app := &application{
		basePath: basePath,
		logger:   logger,
		repo:     repository.New(db),
		cookies:  cookies,
		jwt:      jwt,
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

	logger.Info("gateway online", slog.String("port", port), slog.String("base path", basePath))
	logger.Info("app available at http://localhost" + port + basePath + "/status")

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
