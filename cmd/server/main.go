package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/middleware"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	var handler slog.Handler = slog.NewJSONHandler(os.Stderr, nil)
	if config.Development() {
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler)

	sigIntCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cookies, err := config.NewCookies()
	if err != nil {
		logger.Error("failed to read cookies config", slog.Any("error", err))
		os.Exit(1)
	}
	jwt, err := config.NewJWT()
	if err != nil {
		logger.Error("failed to read jwt config", slog.Any("error", err))
		os.Exit(1)
	}
	ws, err := config.NewWebSocket()
	if err != nil {
		logger.Error("failed to read ws config", slog.Any("error", err))
		os.Exit(1)
	}
	port := config.Port()

	dbURL, err := config.DbURL()
	if err != nil {
		logger.Error("invalid database config", slog.Any("error", err))
		os.Exit(1)
	}
	db, err := pgxpool.New(sigIntCtx, dbURL)
	if err != nil {
		logger.Error("failed to create pgxpool", slog.Any("error", err))
		os.Exit(1)
	}
	_, err = backoff.Retry(
		sigIntCtx, func() (any, error) {
			_, err := db.Query(sigIntCtx, "SELECT 1")
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.SQLState()[:2] == "28" {
					return nil, backoff.Permanent(err)
				}
				logger.Warn("failed to query database", slog.Any("error", err))
			}
			return nil, err
		}, backoff.WithBackOff(backoff.NewExponentialBackOff()), backoff.WithMaxTries(5),
	)
	if err != nil {
		logger.Error("database unavailable", slog.Any("error", err))
		os.Exit(1)
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		logger.Error("unable to create migrations iofs", slog.Any("error", err))
		os.Exit(1)
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		logger.Error("unable to create migrator", slog.Any("error", err))
		os.Exit(1)
	}
	version, dirty, err := migrator.Version()
	if err != nil {
		logger.Error("failed to check migration version", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("migration successful", slog.Uint64("version", uint64(version)), slog.Bool("dirty", dirty))

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
	logger.Info("minesweeper server listening", slog.Any("url", fmt.Sprintf("http://localhost%s", port)))

	select {
	case <-sigIntCtx.Done():
		break
	case err := <-errCh:
		logger.Error("failed to start", slog.Any("error", err))
		os.Exit(1)
	}

	sCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	server.Shutdown(sCtx)
}
