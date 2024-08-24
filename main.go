package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/cors"

	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
)

var port int

func init() {
	const (
		defaultPort = 8000
		usage       = "listening port"
	)
	flag.IntVar(&port, "port", defaultPort, usage)
	flag.IntVar(&port, "p", defaultPort, usage+" (shorthand)")
}

var log = logrus.New()

func init() {
	log.SetLevel(logrus.DebugLevel)
	mines.Log.SetLevel(logrus.DebugLevel)
}

var pg *postgres

func main() {
	flag.Parse()
	log.Info("starting up")

	dbUrl := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)
	var err error
	pg, err = NewPostgres(context.Background(), dbUrl)
	if err != nil {
		log.Fatal("unable to create connection pool: ", err)
	}
	defer pg.Close()
	if err := pg.Ping(context.Background()); err != nil {
		log.Fatal("unable to ping database: ", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/status", handleStatus)
	mux.HandleFunc("GET /v1/records", handleRecords)

	mux.HandleFunc("POST /v1/game", handleNewGame)
	mux.HandleFunc("GET /v1/game/{id}", handleGetGame)
	mux.HandleFunc("POST /v1/game/{id}/open", handleOpen)
	mux.HandleFunc("POST /v1/game/{id}/flag", handleFlag)
	mux.HandleFunc("POST /v1/game/{id}/chord", handleChord)
	mux.HandleFunc("POST /v1/game/{id}/reveal", handleReveal)
	mux.HandleFunc("POST /v1/game/{id}/batch", handleBatch)

	mux.HandleFunc("/v1/game/{id}/connect", handleConnectWs)

	h := useMiddleware(mux,
		loggingMiddleware,
		cors.Default().Handler,
	)

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Infof("ready to serve @ %s", addr)

	log.Fatal(http.ListenAndServe(addr, h))
}
