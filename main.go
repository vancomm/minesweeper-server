package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/schema"
)

type QueryParams struct {
	Height int `schema:"height"`
	Width  int `schema:"width"`
	Bombs  int `schema:"bombs"`
}

var decoder = schema.NewDecoder()

func handleGetField(w http.ResponseWriter, r *http.Request) {
	var query QueryParams
	if err := decoder.Decode(&query, r.URL.Query()); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}
	response := fmt.Sprintf("query: %v\n", query)
	log.Print(response)
	fmt.Fprint(w, response)
}

func main() {

	router := http.NewServeMux()

	router.HandleFunc("GET /field", handleGetField)

	log.Fatal(http.ListenAndServe(":8080", router))
}
