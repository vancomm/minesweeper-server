package main

import (
	"context"
	"embed"
	"errors"
	"hash/maphash"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/database"
	"github.com/vancomm/minesweeper-server/internal/middleware"
)

//go:embed migrations/*.sql
var migrations embed.FS

func createRand() *rand.Rand {
	return rand.New(rand.NewPCG(
		new(maphash.Hash).Sum64(), new(maphash.Hash).Sum64(),
	))
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	_, err := database.ConnectAndMigrate(ctx, migrations)
	if err != nil {
		logger.Error("failed to connect and migrate db", "error", err)
	}

	db, err := database.Connect(ctx)
	if err != nil {
		logger.Error("failed to connect to db", "error", err)
		return
	}

	ws, err := config.NewWebSocket()
	if err != nil {
		logger.Error("failed to read ws config", "error", err)
		return
	}

	game := NewGameHandler(logger, db, ws, createRand())

	router := http.NewServeMux()
	router.HandleFunc("POST /games", game.NewGame)
	router.HandleFunc("GET /games/{id}", game.Fetch)
	router.HandleFunc("POST /games/{id}/move", game.MakeAMove)
	router.HandleFunc("POST /games/{id}/forfeit", game.Forfeit)
	router.HandleFunc("/games/{id}/connect", game.ConnectWS)

	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.Wrap(router, middleware.Logging(logger)),
	}

	done := make(chan struct{})
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to listen and serve", slog.Any("error", err))
		}
		close(done)
	}()

	logger.Info("game server listening", slog.String("addr", ":8080"))
	select {
	case <-done:
		break
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		server.Shutdown(ctx)
		cancel()
	}
}
