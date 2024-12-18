package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vancomm/minesweeper-server/internal/config"
)

func loadConfigFromURL() (*pgxpool.Config, error) {
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		return nil, fmt.Errorf("no DATABASE_URL env variable set")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DATABASE_URL: %w", err)
	}
	return config, nil
}

func loadConfig() (*pgxpool.Config, error) {
	cfg, err := config.NewDatabase()
	if err == nil {
		return pgxpool.ParseConfig(cfg.DSN())
	}

	pgCfg, _err := loadConfigFromURL()
	if _err != nil {
		return nil, errors.Join(err, _err)
	}
	return pgCfg, nil
}

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return pgxpool.NewWithConfig(ctx, config)

}

func ConnectAndMigrate(ctx context.Context, migrations fs.FS) (*pgxpool.Pool, error) {
	conn, err := Connect(ctx)
	if err != nil {
		return nil, err
	}

	url, err := dbURL()
	if err != nil {
		return nil, err
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("unable to create migrations iofs: %w", err)
	}

	migrator, err := migrate.NewWithSourceInstance("iofs", source, url)
	if err != nil {
		return nil, fmt.Errorf("unable to create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return conn, nil
}

func dbURL() (string, error) {
	cfg, err := config.NewDatabase()
	if err != nil {
		return "", err
	}

	return cfg.URL(), err
}
