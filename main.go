package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/schema"
)

type QueryParams struct {
	Height int `schema:"height,required"`
	Width  int `schema:"width,required"`
	Bombs  int `schema:"bombs,required"`
}

var schemaDecoder = schema.NewDecoder()

func sendJson(w http.ResponseWriter, p any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(p)
}

func handleGetField(w http.ResponseWriter, r *http.Request) {
	var query QueryParams
	if err := schemaDecoder.Decode(&query, r.URL.Query()); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if merr, ok := err.(schema.MultiError); ok {
			errPayload := make(map[string]string)
			for k, v := range merr {
				errPayload[k] = v.Error()
			}
			payload := make(map[string]any)
			payload["errors"] = errPayload
			sendJson(w, payload)
		} else {
			log.Printf("not ok, err: %v\n", err)
		}
		return
	}
	field := CreateSolvableField(query.Height, query.Width, query.Bombs)
	sendJson(w, field)
}

func main() {
	schemaDecoder.IgnoreUnknownKeys(true)
	router := http.NewServeMux()

	router.HandleFunc("GET /field", handleGetField)

	log.Fatal(http.ListenAndServe(":8080", router))
}
