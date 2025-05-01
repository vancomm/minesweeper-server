package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vancomm/minesweeper-server/internal/config"
)

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := config.NewPgxpoolConfig()
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, config)

}

func Migrate(url string, migrations fs.FS) (migrator *migrate.Migrate, err error) {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("unable to create migrations iofs: %w", err)
	}
	migrator, err = migrate.NewWithSourceInstance("iofs", source, url)
	if err != nil {
		return nil, fmt.Errorf("unable to create migrator: %w", err)
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	return migrator, nil
}

func ConnectAndMigrate(ctx context.Context, migrations fs.FS) (*pgxpool.Pool, *migrate.Migrate, error) {
	conn, err := Connect(ctx)
	if err != nil {
		return nil, nil, err
	}
	url, err := config.DbURL()
	if err != nil {
		return nil, nil, err
	}
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create migrations iofs: %w", err)
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", source, url)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create migrator: %w", err)
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	return conn, migrator, nil
}
