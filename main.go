package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
)

var _, development = os.LookupEnv("DEVELOPMENT")

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

// This endpoint may be called for the side effect in [authMiddleware] that
// clears expired auth cookies
func handleStatus(w http.ResponseWriter, r *http.Request) {
	if claims, ok := r.Context().Value(ctxKeyPlayerClaims).(*PlayerClaims); ok {
		w.Write([]byte("you are authenticated as " + claims.Username))
	} else {
		w.Write([]byte("\"ok\""))
	}
}

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

	mux.HandleFunc("POST /v1/register", handleRegister)
	mux.HandleFunc("POST /v1/login", handleLogin)
	mux.HandleFunc("POST /v1/logout", handleLogout)

	mux.HandleFunc("GET /v1/status", handleStatus)
	mux.HandleFunc("GET /v1/records", handleRecords)
	mux.HandleFunc("GET /v1/myrecords", handlePlayerRecords)

	mux.HandleFunc("POST /v1/game", handleNewGame)
	mux.HandleFunc("GET /v1/game/{id}", handleGetGame)
	mux.HandleFunc("POST /v1/game/{id}/open", handleOpen)
	mux.HandleFunc("POST /v1/game/{id}/flag", handleFlag)
	mux.HandleFunc("POST /v1/game/{id}/chord", handleChord)
	mux.HandleFunc("POST /v1/game/{id}/reveal", handleReveal)
	mux.HandleFunc("POST /v1/game/{id}/batch", handleBatch)

	mux.HandleFunc("/v1/game/{id}/connect", handleConnectWs)

	h := useMiddleware(mux,
		corsMiddleware,
		authMiddleware,
		loggingMiddleware,
		// cors.AllowAll().Handler,
		// corsMiddleware,
	)

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Infof("ready to serve @ %s", addr)

	log.Fatal(http.ListenAndServe(addr, h))
}
