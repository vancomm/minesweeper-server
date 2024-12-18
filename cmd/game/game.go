package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/handlers"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

type GameHandler struct {
	logger *slog.Logger
	repo   *repository.Queries
	ws     *config.WebSocket
	rnd    *rand.Rand
}

func NewGameHandler(
	logger *slog.Logger,
	db *pgxpool.Pool,
	ws *config.WebSocket,
	rnd *rand.Rand,
) *GameHandler {
	handler := &GameHandler{
		logger: logger,
		repo:   repository.New(db),
		ws:     ws,
		rnd:    rnd,
	}
	return handler
}

func getAuthenticatedPlayerId(r *http.Request) (int64, bool) {
	playerIdStr := r.Header.Get("X-Player-ID")
	playerId, err := strconv.ParseInt(playerIdStr, 10, 64)
	return playerId, err == nil
}

func (g GameHandler) NewGame(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	dto, err := ParseNewGameDTO(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, g.logger, err)
		return
	}

	gameParams := mines.GameParams(dto)

	pos, err := ParsePosition(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, g.logger, err)
		return
	}

	if !gameParams.ValidatePosition(pos.X, pos.Y) {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, g.logger, fmt.Errorf("invalid cell position"))
		return
	}

	game, err := mines.NewGame(&gameParams, pos.X, pos.Y, g.rnd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("unable to generate a new game", "error", err)
		return
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(game); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("unable to encode game state", "error", err)
		return
	}

	playerId, ok := getAuthenticatedPlayerId(r)
	if ok {
		g.logger.Debug("creating player session", "playerId", playerId)

		session, err := g.repo.CreatePlayerSession(
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
			g.logger.Error("unable to create player game session", "error", err)
			return
		}

		handlers.SendJSONOrLog(w, g.logger, session)
	} else {
		g.logger.Debug("creating anonymous session")

		session, err := g.repo.CreateAnonymousSession(
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
			g.logger.Error("unable to create anonymous game session", "error", err)
			return
		}

		handlers.SendJSONOrLog(w, g.logger, session)
	}

}

func (g GameHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	session, err := g.repo.GetSession(r.Context(), sessionId)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("unable to fetch session from db", "error", err)
		return
	}

	playerId, ok := getAuthenticatedPlayerId(r)
	if ok && session.PlayerID != nil && *session.PlayerID != playerId {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	handlers.SendJSONOrLog(w, g.logger, session)
}

func (g GameHandler) MakeAMove(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	move, err := ParseGameMove(query.Get("move"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, g.logger, err)
		return
	}

	pos, err := ParsePosition(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handlers.SendErrorOrLog(w, g.logger, err)
		return
	}

	sessionId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	session, err := g.repo.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("could not fetch session from db", "error", err)
		return
	}

	playerId, ok := getAuthenticatedPlayerId(r)
	if ok && session.PlayerID != nil && *session.PlayerID != playerId {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	game, err := mines.ParseGameStateFromBytes(session.State)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("db returned invalid game_session.state", "error", err)
		return
	}

	if !game.ValidatePosition(pos.X, pos.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch move {
	case Open:
		game.OpenCell(pos.X, pos.Y)
	case Flag:
		game.FlagCell(pos.X, pos.Y)
	case Chord:
		game.ChordCell(pos.X, pos.Y)
	}

	if game.Won || game.Dead {
		game.RevealMines()
		*session.EndedAt = time.Now().UTC()
	}

	b, err := game.Bytes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("unable to serialize game state", "error", err)
		return
	}

	err = g.repo.UpdateSession(r.Context(), repository.UpdateSessionParams{
		GameSessionID: session.GameSessionID,
		State:         b,
		Dead:          game.Dead,
		Won:           game.Won,
		EndedAt:       session.EndedAt,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("unable to update session in db", "error", err)
		return
	}

	handlers.SendJSONOrLog(w, g.logger, session)
}

func (g GameHandler) Forfeit(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	session, err := g.repo.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("could not fetch session from db", "error", err)
		return
	}

	playerId, ok := getAuthenticatedPlayerId(r)
	if ok && session.PlayerID != nil && *session.PlayerID != playerId {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	game, err := mines.ParseGameStateFromBytes(session.State)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	game.RevealAll()

	if session.EndedAt == nil {
		*session.EndedAt = time.Now().UTC()
	}

	b, err := game.Bytes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("unable to serialize game state", "error", err)
		return
	}

	err = g.repo.UpdateSession(r.Context(), repository.UpdateSessionParams{
		GameSessionID: session.GameSessionID,
		State:         b,
		Dead:          game.Dead,
		Won:           game.Won,
		EndedAt:       session.EndedAt,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("unable to update session in db", "error", err)
		return
	}

	handlers.SendJSONOrLog(w, g.logger, session)
}
