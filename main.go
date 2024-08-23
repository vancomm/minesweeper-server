package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
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

var (
	log = logrus.New()
	kvs *Store
	pg  *postgres
)

func init() {
	log.SetLevel(logrus.DebugLevel)
	mines.Log.SetLevel(logrus.DebugLevel)
}

func main() {
	flag.Parse()
	log.Info("starting up")

	var err error
	pg, err = NewPostgres(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("unable to create connection pool: ", err)
	}
	defer pg.Close()

	db, err := sql.Open("sqlite3", "./data/sessions.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	defer db.Close()

	kvs, err = NewStore(db, "sessions")
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/status", handleStatus)
	mux.HandleFunc("POST /v1/game", handleNewGame)
	mux.HandleFunc("GET /v1/game/{id}", handleGetState)
	mux.HandleFunc("POST /v1/game/{id}/open", handleOpen)
	mux.HandleFunc("POST /v1/game/{id}/flag", handleFlag)
	mux.HandleFunc("POST /v1/game/{id}/chord", handleChord)
	mux.HandleFunc("POST /v1/game/{id}/reveal", handleReveal)
	mux.HandleFunc("POST /v1/game/{id}/batch", handleBatch)

	mux.HandleFunc("GET /v1/records", handleRecords)

	mux.HandleFunc("/v1/game/{id}/connect", handleConnectWs)

	mux.HandleFunc("POST /v2/game", handlePostGame)

	h := useMiddleware(mux,
		loggingMiddleware,
		cors.Default().Handler,
	)

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Infof("ready to serve @ %s", addr)

	log.Fatal(http.ListenAndServe(addr, h))
}
