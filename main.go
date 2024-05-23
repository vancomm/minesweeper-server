package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

var requiredParams = []string{"w", "h", "b"}

func handleGetField(w http.ResponseWriter, r *http.Request) {
	var buf strings.Builder
	buf.WriteString("your search params: ")
	for _, paramName := range requiredParams {
		param := r.URL.Query().Get(paramName)
		if param == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "missing param: %s", paramName)
			return
		}
		fmt.Fprintf(&buf, "%s: %s, ", paramName, param)
	}
	response := strings.TrimSuffix(buf.String(), ", ")
	fmt.Fprintln(w, response)
}

func main() {
	router := http.NewServeMux()

	router.HandleFunc("GET /field", handleGetField)

	log.Fatal(http.ListenAndServe(":8080", router))
}
