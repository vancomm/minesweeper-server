package app

import (
	"hash/maphash"
	"math/rand/v2"

	"github.com/vancomm/minesweeper-server/internal/handlers"
)

func createRand() *rand.Rand {
	return rand.New(rand.NewPCG(
		new(maphash.Hash).Sum64(), new(maphash.Hash).Sum64(),
	))
}

func (a *App) loadRoutes() {
	game := handlers.NewGameHandler(
		a.logger, a.db, a.cookies, a.ws, createRand(),
	)

	a.router.HandleFunc("POST /game", game.NewGame)
	a.router.HandleFunc("GET /game/{id}", game.Fetch)
	a.router.HandleFunc("POST /game/{id}/move", game.MakeAMove)
	a.router.HandleFunc("POST /game/{id}/forfeit", game.Forfeit)
	a.router.HandleFunc("/game/{id}/connect", game.ConnectWS)
}
