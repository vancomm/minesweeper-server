package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vancomm/minesweeper-server/internal/mines"
)

type postgres struct {
	db *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dbUrl string) (*postgres, error) {
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, err
	}
	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
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

func (pg *postgres) CreatePlayer(
	ctx context.Context, username string, passwordHash []byte,
) (*Player, error) {
	var playerId int
	if err := pg.db.QueryRow(ctx, `
		INSERT INTO player (
			username, password_hash
		) 
		VALUES (
			@username, @password_hash
		)
		RETURNING player_id`,
		pgx.NamedArgs{
			"username":      username,
			"password_hash": passwordHash,
		}).Scan(&playerId); err != nil {
		return nil, err
	}
	player := &Player{
		PlayerId: playerId,
		Username: username,
	}
	return player, nil
}

func (pg *postgres) GetPlayer(
	ctx context.Context, username string,
) (*Player, error) {
	rows, err := pg.db.Query(ctx, `
		SELECT player_id, username, password_hash
		FROM player
		WHERE username = $1;`,
		username)
	if err != nil {
		return nil, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Player])
}

func (pg *postgres) CreateAnonymousGameSession(
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

func (pg *postgres) CreatePlayerGameSession(
	ctx context.Context, playerId int, state *mines.GameState,
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
			player_id, width, height, mine_count, "unique", dead, won, state
		) 
		VALUES (
			@player_id, @width, @height, @mine_count, @unique, @dead, @won, @state
		) 
		RETURNING game_session_id, started_at;`,
		pgx.NamedArgs{
			"player_id":  playerId,
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
