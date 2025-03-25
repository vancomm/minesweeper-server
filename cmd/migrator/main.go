package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lmittmann/tint"
	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/database"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	var logger *slog.Logger
	if config.Development() {
		logger = slog.New(tint.NewHandler(os.Stderr, nil))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	_, migrator, err := database.ConnectAndMigrate(ctx, migrations)
	if err != nil {
		logger.Error("failed to connect to db", slog.Any("error", err))
		os.Exit(1)
	}
	version, dirty, err := migrator.Version()
	if err != nil {
		logger.Error("failed to check migration version", slog.Any("error", err))
	} else {
		logger.Info("migration successful", slog.Uint64("version", uint64(version)), slog.Bool("dirty", dirty))
	}
	os.Exit(0)
}
