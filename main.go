package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/schema"
)

var requiredParams = []string{"w", "h", "b"}

type FieldParams struct {
	height int
	width  int
	bombs  int
}

var decoder = schema.NewDecoder()

func parseFieldParams(r *http.Request, paramName string) (int, error) {
	var (
		param int
		err   error
	)

	rawParam := r.URL.Query().Get(paramName)
	if rawParam == "" {
		err = fmt.Errorf("missing param %s", paramName)
		return param, err
	}

	if _, err := fmt.Sscan(rawParam, &param); err != nil {
		err = fmt.Errorf("invalid param %s: must be integer", paramName)
		return param, err
	}

	return param, err
}

func handleGetField(w http.ResponseWriter, r *http.Request) {
	var buf strings.Builder
	buf.WriteString("your search params: ")

	for _, paramName := range requiredParams {
		param, err := parseFieldParams(r, paramName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, err)
		}
		fmt.Fprintf(&buf, "%s: %d", paramName, param)
	}

	response := strings.TrimSuffix(buf.String(), ", ")
	fmt.Fprintln(w, response)
}

func main() {
	router := http.NewServeMux()

	router.HandleFunc("GET /field", handleGetField)

	log.Fatal(http.ListenAndServe(":8080", router))
}
