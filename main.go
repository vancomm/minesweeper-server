package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/schema"
	"github.com/sirupsen/logrus"
)

var (
	log           *logrus.Logger
	schemaDecoder *schema.Decoder
)

func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = new(logrus.JSONFormatter)
	file, err := os.OpenFile("logs.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		log.Warn("failed to log to file, using stderr")
	} else {
		log.Out = io.MultiWriter(file, os.Stdout)
	}

	schemaDecoder = schema.NewDecoder()
	schemaDecoder.IgnoreUnknownKeys(true)
}

func sendJson(w http.ResponseWriter, p any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(p)
}

func handleGetBoard(w http.ResponseWriter, r *http.Request) {
	var board Board
	if err := schemaDecoder.Decode(&board, r.URL.Query()); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		var (
			payload = make(map[string]any)
			errors  = make(map[string]string)
		)
		payload["errors"] = errors
		if merr, ok := err.(schema.MultiError); ok {
			for k, v := range merr {
				errors[k] = v.Error()
			}
		} else {
			errors["schema"] = err.Error()
		}
		if err := sendJson(w, payload); err != nil {
			log.Fatal(err)
		}
		return
	}

	if ok := board.Solvable(); !ok {
		w.WriteHeader(http.StatusInternalServerError)
		log.WithField("board", board).Fatal("could not find a solvable board")
	}

	var payload = board.Cells()
	log.WithField("payload", payload).Debug("payload")

	if err := sendJson(w, payload); err != nil {
		log.WithFields(logrus.Fields{
			"board":   board,
			"payload": payload,
		}).Fatal(err)
	}
}

func LoggerMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(logrus.Fields{
			"uri": r.RequestURI,
		}).Info("new request")
		next.ServeHTTP(w, r)
	}
}

func main() {
	log.Info("starting server")
	var router = http.NewServeMux()
	router.HandleFunc("GET /board", handleGetBoard)
	var handler = LoggerMiddleware(router)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
