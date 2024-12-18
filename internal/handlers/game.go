package handlers

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

	"github.com/gorilla/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vancomm/minesweeper-server/internal/config"
	"github.com/vancomm/minesweeper-server/internal/middleware"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

type GameHandler struct {
	logger  *slog.Logger
	repo    *repository.Queries
	cookies *config.Cookies
	ws      *config.WebSocket
	rnd     *rand.Rand
}

func NewGameHandler(
	logger *slog.Logger,
	db *pgxpool.Pool,
	cookies *config.Cookies,
	ws *config.WebSocket,
	rnd *rand.Rand,
) *GameHandler {
	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)

	handler := &GameHandler{
		logger:  logger,
		repo:    repository.New(db),
		cookies: cookies,
		ws:      ws,
		rnd:     rnd,
	}

	return handler
}

func (g GameHandler) NewGame(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	dto, err := ParseCreateNewGameDTO(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		sendJSONOrLog(w, g.logger, wrapError(err))
		return
	}

	gameParams := mines.GameParams(dto)

	pos, err := ParsePosition(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		sendJSONOrLog(w, g.logger, wrapError(err))
		return
	}

	if !gameParams.ValidatePosition(pos.X, pos.Y) {
		w.WriteHeader(http.StatusBadRequest)
		sendJSONOrLog(w, g.logger, wrapError(fmt.Errorf("invalid cell position")))
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

	claims, loggedIn := (r.Context().Value(middleware.CtxPlayerClaims).(*config.PlayerClaims))

	if loggedIn {
		g.logger.Debug("creating player session", "claims", claims)

		session, err := g.repo.CreatePlayerSession(
			r.Context(), repository.CreatePlayerSessionParams{
				PlayerID:  &claims.PlayerId,
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

		sendJSONOrLog(w, g.logger, session)
		return
	}

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

	sendJSONOrLog(w, g.logger, NewGameSessionDTO(
		session.GameSessionID, session.StartedAt, nil, game,
	))
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

	game, err := mines.ParseGameStateFromBytes(session.State)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("db returned invalid game_session.state", "error", err)
		return
	}

	sendJSONOrLog(w, g.logger, NewGameSessionDTO(
		session.GameSessionID, session.StartedAt, session.EndedAt, game,
	))
}

func (g GameHandler) MakeAMove(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	move, err := ParseGameMove(query.Get("move"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		sendJSONOrLog(w, g.logger, wrapError(err))
		return
	}

	pos, err := ParsePosition(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		sendJSONOrLog(w, g.logger, wrapError(err))
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

	sendJSONOrLog(w, g.logger, NewGameSessionDTO(
		session.GameSessionID, session.StartedAt, session.EndedAt, game,
	))
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

	sendJSONOrLog(w, g.logger, NewGameSessionDTO(
		session.GameSessionID, session.StartedAt, session.EndedAt, game,
	))
}
