package main

import (
	"net/http"

	"github.com/vancomm/minesweeper-server/mines"
)

func handleGetRecords(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	options := []GameRecordsOption{}
	if query.Has("username") {
		options = append(options, GameRecordsForPlayer(query.Get("username")))
	}
	if query.Has("seed") {
		gameParams, err := mines.ParseGameParams(query.Get("seed"))
		if err != nil {
			log.Debug(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		options = append(options, GameRecordsForGameParams(gameParams))
	}
	records, err := getGameRecords(r.Context(), options...)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := sendJSON(w, records); err != nil {
		log.Error(err)
	}
}

// func handleRecords(w http.ResponseWriter, r *http.Request) {
// 	records, err := compileGameRecords(r.Context())
// 	if err != nil {
// 		log.Error(err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
// 	if _, err := sendJSON(w, records); err != nil {
// 		log.Error(err)
// 	}
// }

func handleGetOwnRecords(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(ctxPlayerClaims).(*PlayerClaims)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	records, err := getGameRecords(
		r.Context(), GameRecordsForPlayer(claims.Username),
	)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := sendJSON(w, records); err != nil {
		log.Error(err)
	}
}
