package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vancomm/minesweeper-server/mines"
)

const (
	createPlayerTable = `
CREATE TABLE IF NOT EXISTS player (
	player_id 		bigint 	GENERATED ALWAYS AS IDENTITY 
							PRIMARY KEY,
	username 		text 	UNIQUE NOT NULL,
	password_hash 	bytea 	NOT NULL,
	created_at 		timestamp with time zone
							DEFAULT now() 
							NOT NULL,
	updated_at 		timestamp with time zone 
							DEFAULT now() 
							NOT NULL
);`
	createGameSessionTable = `
CREATE TABLE IF NOT EXISTS game_session (
	game_session_id	bigint 	GENERATED ALWAYS AS IDENTITY 
							PRIMARY KEY,
	player_id		bigint	REFERENCES player (player_id)
							NULL,
	width			integer	NOT NULL,
	height			integer	NOT NULL,
	mine_count		integer	NOT NULL,
	"unique"		boolean NOT NULL,
	dead			boolean NOT NULL,
	won				boolean NOT NULL,
	started_at		timestamp with time zone
							DEFAULT now()
							NOT NULL,
	ended_at		timestamp with time zone
							NULL,
	state			bytea	NOT NULL,
	created_at 		timestamp with time zone
							DEFAULT now() 
							NOT NULL,
	updated_at 		timestamp with time zone 
							DEFAULT now() 
							NOT NULL
	);`
	createUpdateModifiedColumnFunction = `
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = now();
	RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';`
	createPlayerUpdateTrigger = `
CREATE OR REPLACE TRIGGER update_players_modtime
BEFORE UPDATE ON player
FOR EACH ROW EXECUTE FUNCTION update_modified_column();`
	createGameSessionUpdateTrigger = `
CREATE OR REPLACE TRIGGER update_game_session_modtime
BEFORE UPDATE ON game_session
FOR EACH ROW EXECUTE FUNCTION update_modified_column();`
	initSql = createPlayerTable +
		createGameSessionTable +
		createUpdateModifiedColumnFunction +
		createPlayerUpdateTrigger +
		createGameSessionUpdateTrigger
)

type postgres struct {
	db *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dbUrl string) (*postgres, error) {
	dbconfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, err
	}

	db, err := pgxpool.NewWithConfig(ctx, dbconfig)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(ctx, initSql); err != nil {
		return nil, err
	}
	return &postgres{db}, nil
}

func (pg *postgres) Close() {
	pg.db.Close()
}

func (pg *postgres) CreateSession(ctx context.Context, state *mines.GameState) (*GameSession, error) {
	var (
		stateBuf  bytes.Buffer
		sessionId int
		startedAt time.Time
	)
	enc := gob.NewEncoder(&stateBuf)
	if err := enc.Encode(state); err != nil {
		return nil, err
	}
	row := pg.db.QueryRow(ctx, `
	INSERT INTO game_session (
		width, height, mine_count, 
		"unique", dead, won, state
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING game_session_id, started_at;`,
		state.Width, state.Height, state.MineCount,
		state.Unique, state.Dead, state.Won, stateBuf.Bytes())
	if err := row.Scan(&sessionId, &startedAt); err != nil {
		return nil, err
	}
	session := &GameSession{
		SessionId: strconv.Itoa(sessionId),
		State:     *state,
		StartedAt: startedAt,
	}
	return session, nil
}

func handlePostGame(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var (
		gameParams NewGameParams
		posParams  PosParams
	)
	if err := dec.Decode(&gameParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := dec.Decode(&posParams, query); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	params := mines.GameParams(gameParams)
	if !params.ValidateSquare(posParams.X, posParams.Y) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	game, err := mines.New(params, posParams.X, posParams.Y, rnd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	session, err := pg.CreateSession(context.Background(), game)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
	}
	if err := sendJSON(w, session); err != nil {
		log.Error(err)
	}
}
