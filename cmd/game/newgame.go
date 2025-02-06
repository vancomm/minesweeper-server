package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"

	"github.com/vancomm/minesweeper-server/internal/handlers"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleNewGame(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	dto, err := ParseNewGame(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, err)
		return
	}

	gameParams := mines.GameParams(dto)

	p, err := parsePoint(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, err)
		return
	}

	if !gameParams.ValidatePoint(p.X, p.Y) {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, app.logger, fmt.Errorf("invalid cell point"))
		return
	}

	game, err := mines.NewGame(&gameParams, p.X, p.Y, app.rnd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("unable to generate a new game", "error", err)
		return
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(game); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		app.logger.Error("unable to encode game state", "error", err)
		return
	}

	playerId, ok := getAuthenticatedPlayerId(r)
	if ok {
		app.logger.Debug("creating player session", "playerId", playerId)

		session, err := app.repo.CreatePlayerSession(
			r.Context(), repository.CreatePlayerSessionParams{
				PlayerID:  &playerId,
				Width:     int32(game.Width),
				Height:    int32(game.Height),
				MineCount: int32(game.MineCount),
				Unique:    game.Unique,
				Dead:      game.Dead,
				Won:       game.Won,
				State:     buf.Bytes(),
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			app.logger.Error("unable to create player game session", "error", err)
			return
		}

		handlers.SendJSONOrLog(w, app.logger, session)
	} else {
		app.logger.Debug("creating anonymous session")

		session, err := app.repo.CreateAnonymousSession(
			r.Context(),
			repository.CreateAnonymousSessionParams{
				Width:     int32(game.Width),
				Height:    int32(game.Height),
				MineCount: int32(game.MineCount),
				Unique:    game.Unique,
				Dead:      game.Dead,
				Won:       game.Won,
				State:     buf.Bytes(),
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			app.logger.Error("unable to create anonymous game session", "error", err)
			return
		}

		handlers.SendJSONOrLog(w, app.logger, session)
	}

}
