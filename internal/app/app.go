package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/database"
	"github.com/vancomm/minesweeper-server/internal/middleware"
)

type App struct {
	logger     *slog.Logger
	router     *http.ServeMux
	db         *pgxpool.Pool
	cookies    *config.Cookies
	ws         *config.WebSocket
	migrations fs.FS
}

func New(logger *slog.Logger, migrations fs.FS) *App {
	router := http.NewServeMux()

	app := &App{
		logger:     logger,
		router:     router,
		migrations: migrations,
	}

	return app
}

func (a *App) Start(ctx context.Context) error {
	db, err := database.ConnectAndMigrate(ctx, a.migrations)
	if err != nil {
		return fmt.Errorf("unable to connect to db: %w", err)
	}

	a.db = db

	cookies, err := config.NewCookies()
	if err != nil {
		return err
	}

	a.cookies = cookies

	ws, err := config.NewWebSocket()
	if err != nil {
		return err
	}

	a.ws = ws

	a.loadRoutes()

	server := &http.Server{
		Addr: ":8080",
		Handler: middleware.Wrap(
			a.router,
			middleware.Cors(),
			middleware.Auth(a.logger, cookies),
		),
	}

	done := make(chan struct{})
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("unable to listen and serve", slog.Any("error", err))
		}
		close(done)
	}()

	a.logger.Info("server listening", slog.String("addr", ":8080"))
	select {
	case <-done:
		break
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		server.Shutdown(ctx)
		cancel()
	}

	return nil
}
