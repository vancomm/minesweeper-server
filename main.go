package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/schema"
	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Logger
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
}

type session struct {
	board *Board
}

var sessions = make(map[string]*session)

var schemaDecoder *schema.Decoder

func init() {
	schemaDecoder = schema.NewDecoder()
	schemaDecoder.IgnoreUnknownKeys(true)
}

func sendJson(w http.ResponseWriter, p any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(p)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	var (
		token = uuid.NewString()
		board Board
	)
	err := schemaDecoder.Decode(&board, r.URL.Query())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		var (
			errors  = &map[string]string{}
			payload = map[string]any{
				"errors": errors,
			}
		)
		if merr, ok := err.(schema.MultiError); ok {
			for k, v := range merr {
				(*errors)[k] = v.Error()
			}
		} else {
			(*errors)["schema"] = err.Error()
		}
		if err := sendJson(w, payload); err != nil {
			log.Fatal(err)
		}
		return
	}

	if solved := board.Solvable(1e4); solved {
		sessions[token] = &session{
			board: &board,
		}
		fmt.Fprint(w, token)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not generate a solvable field with passed parameters")
	}
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

	var maxAttempts = 1_000
	if ok := board.Solvable(int(maxAttempts)); !ok {
		var message = fmt.Sprintf("could not find a solvable board in %d attempts", maxAttempts)
		w.WriteHeader(http.StatusInternalServerError)
		var payload = make(map[string]any)
		payload["error"] = message
		sendJson(w, payload)
		log.WithFields(logrus.Fields{
			"board": board,
		}).Error(message)
		return
	}

	var payload = board.Cells()

	if err := sendJson(w, payload); err != nil {
		log.WithFields(logrus.Fields{
			"board":   board,
			"payload": payload,
		}).Fatal(err)
	}
}

func main() {
	log.Info("starting server")
	var router = http.NewServeMux()
	router.HandleFunc("GET /board", handleGetBoard)
	router.HandleFunc("GET /start", handleStart)
	var handler = LoggerMiddleware(router)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func LoggerMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(logrus.Fields{
			"uri": r.RequestURI,
		}).Info("new request")
		next.ServeHTTP(w, r)
	}
}
