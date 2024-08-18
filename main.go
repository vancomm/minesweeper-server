package main

import (
	"database/sql"
	"flag"
	"fmt"
	"hash/maphash"
	"math/rand/v2"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"

	"github.com/gorilla/schema"
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
	dec = schema.NewDecoder()
	r   = rand.New(rand.NewPCG(
		new(maphash.Hash).Sum64(),
		new(maphash.Hash).Sum64(),
	))
	kvs *Store
)

func init() {
	log.SetLevel(logrus.DebugLevel)
	mines.Log.SetLevel(logrus.DebugLevel)

	dec.IgnoreUnknownKeys(true)
}

func main() {
	flag.Parse()
	log.Info("starting up")

	db, err := sql.Open("sqlite3", "./data/sessions.sqlite")
	if err != nil {
		log.Fatalf("failed to connect sqlite db: %v", err)
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

	h := useMiddleware(mux,
		loggingMiddleware,
		cors.Default().Handler,
	)

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Infof("ready to serve @ %s", addr)

	log.Fatal(http.ListenAndServe(addr, h))
}
