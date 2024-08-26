package main

import (
	"encoding/json"
	"iter"
	"net/http"
	"strings"
)

func byPiece(s string, sep string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		i := 0
		found := true
		var piece string
		for found {
			piece, s, found = strings.Cut(s, sep)
			if !yield(i, piece) {
				return
			}
			i += 1
		}
	}
}

func sendJSON(w http.ResponseWriter, v any) (int, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return 0, err
	}
	return w.Write(payload)
}
