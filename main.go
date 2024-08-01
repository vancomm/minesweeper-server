package main

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
)

var (
	log     = logrus.New()
	decoder = schema.NewDecoder()
	r       = rand.New(rand.NewPCG(1, 2))
)

func init() {
	decoder.IgnoreUnknownKeys(true)
}

type NewGameParams struct {
	Width     int  `schema:"width,required"`
	Height    int  `schema:"height,required"`
	MineCount int  `schema:"mine_count,required"`
	Unique    bool `schema:"unique,required"`
}

type OpenParams struct {
	X int `schema:"x,required"`
	Y int `schema:"y,required"`
}

func handleNewGame(w http.ResponseWriter, h *http.Request) {
	var (
		gameParams  NewGameParams
		clickParams OpenParams
		query       = h.URL.Query()
	)

	if err := decoder.Decode(&gameParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := decoder.Decode(&clickParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.WithFields(logrus.Fields{
		"gameParams":  gameParams,
		"clickParams": clickParams,
	}).Info("new game request")

	grid, err := mines.MineGen(
		mines.GameParams(gameParams),
		clickParams.X, clickParams.Y,
		r,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(grid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(j)
}

func handleOpen(w http.ResponseWriter, r *http.Request) {

}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /newgame", handleNewGame)
	mux.HandleFunc("GET /open", handleOpen)

	log.Fatal(http.ListenAndServe("localhost:8000", mux))
}
