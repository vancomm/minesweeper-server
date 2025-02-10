package main

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func (app application) handleMove(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	move, err := decodeGameMove(query.Get("move"))
	if err != nil {
		app.badRequest(w)
		return
	}

	p, err := decodePoint(query)
	if err != nil {
		app.badRequest(w)
		return
	}

	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		app.badRequest(w)
		return
	}

	session, err := app.repo.FetchGameSession(r.Context(), sessionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			app.notFound(w)
		} else {
			app.internalError(w, "could not fetch session from db", slog.Any("error", err))
		}
		return
	}

	playerId, ok := app.getAuthenticatedPlayerId(r)
	if ok && session.PlayerId != nil && *session.PlayerId != playerId {
		app.unauthorized(w)
		return
	}

	game, err := mines.DecodeGameState(session.State)
	if err != nil {
		app.internalError(w, "db returned invalid game_session.state", slog.Any("error", err))
		return
	}

	if !game.PointInBounds(p.X, p.Y) {
		app.badRequest(w)
		return
	}

	switch move {
	case Open:
		game.OpenCell(p.X, p.Y)
	case Flag:
		game.FlagCell(p.X, p.Y)
	case Chord:
		game.ChordCell(p.X, p.Y)
	default:
		app.logger.Warn("unhandled GameMove", slog.Any("move", move))
	}

	if game.Won || game.Dead {
		game.Forfeit()
		session.EndedAt.Time = time.Now().UTC()
	}

	b, err := game.Bytes()
	if err != nil {
		app.internalError(w, "unable to serialize game state", slog.Any("error", err))
		return
	}

	err = app.repo.UpdateGameSession(
		r.Context(),
		session.GameSessionId,
		repository.UpdateGameSessionParams{
			State:   &b,
			Dead:    &game.Dead,
			Won:     &game.Won,
			EndedAt: &session.EndedAt.Time,
		},
	)
	if err != nil {
		app.internalError(w, "unable to update session in db", slog.Any("error", err))
		return
	}

	dto, err := NewGameSessionDTO(*session)
	if err != nil {
		app.internalError(w, "failed to create game session dto", slog.Any("error", err))
		return
	}

	app.replyWith(w, dto)
}
