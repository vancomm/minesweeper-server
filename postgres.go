package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vancomm/minesweeper-server/mines"
)

type postgres struct {
	db *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dbUrl string) (*postgres, error) {
	config, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, err
	}
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return &postgres{db}, nil
}

func (pg *postgres) Ping(ctx context.Context) error {
	return pg.db.Ping(ctx)
}

func (pg *postgres) Close() {
	pg.db.Close()
}

func (pg *postgres) CreateGameSession(
	ctx context.Context, state *mines.GameState,
) (*GameSession, error) {
	var (
		stateBuf      bytes.Buffer
		gameSessionId int
		startedAt     time.Time
	)
	if err := gob.NewEncoder(&stateBuf).Encode(state); err != nil {
		return nil, err
	}
	if err := pg.db.QueryRow(ctx, `
		INSERT INTO game_session (
			width, height, mine_count, "unique", dead, won, state
		)
		VALUES (
			@width, @height, @mine_count, @unique, @dead, @won, @state
		)
		RETURNING game_session_id, started_at;`,
		pgx.NamedArgs{
			"width":      state.Width,
			"height":     state.Height,
			"mine_count": state.MineCount,
			"unique":     state.Unique,
			"dead":       state.Dead,
			"won":        state.Won,
			"state":      stateBuf.Bytes(),
		}).Scan(&gameSessionId, &startedAt); err != nil {
		return nil, err
	}
	session := &GameSession{
		SessionId: gameSessionId,
		State:     *state,
		StartedAt: startedAt,
	}
	return session, nil
}

func (pg *postgres) GetSession(
	ctx context.Context, gameSessionId int,
) (*GameSession, error) {
	var (
		stateBuf  []byte
		state     mines.GameState
		startedAt time.Time
		endedAt   pgtype.Timestamptz
	)
	if err := pg.db.QueryRow(ctx, `
		SELECT state, started_at, ended_at
		FROM game_session
		WHERE game_session_id = $1;`,
		gameSessionId).Scan(
		&stateBuf, &startedAt, &endedAt,
	); err != nil {
		return nil, err
	}
	if err := gob.NewDecoder(bytes.NewBuffer(stateBuf)).Decode(&state); err != nil {
		return nil, err
	}
	gameSession := &GameSession{
		SessionId: gameSessionId,
		State:     state,
		StartedAt: startedAt,
		EndedAt:   endedAt.Time,
	}
	return gameSession, nil
}

func (pg *postgres) UpdateGameSession(
	ctx context.Context, gameSession *GameSession,
) error {
	var stateBuf bytes.Buffer
	if err := gob.NewEncoder(&stateBuf).Encode(gameSession.State); err != nil {
		return err
	}
	_, err := pg.db.Exec(ctx, `
		UPDATE game_session
		SET dead = @dead
			, won = @won
			, ended_at = @ended_at
			, state = @state
		WHERE game_session_id = @game_session_id;`,
		pgx.NamedArgs{
			"game_session_id": gameSession.SessionId,
			"dead":            gameSession.State.Dead,
			"won":             gameSession.State.Won,
			"ended_at":        gameSession.EndedAt,
			"state":           stateBuf.Bytes(),
		})
	return err
}