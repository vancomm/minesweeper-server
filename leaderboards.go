package main

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type GameRecord struct {
	Username  *string `json:"username"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	MineCount int     `json:"mine_count"`
	Unique    bool    `json:"unique"`
	Playtime  float64 `json:"playtime"`
}

func compileGameRecords(ctx context.Context) ([]GameRecord, error) {
	rows, err := pg.db.Query(ctx, `
	SELECT username 
		, width
		, height
		, mine_count
		, "unique"
		, extract(epoch from (ended_at - started_at)) playtime
	FROM game_session
	LEFT OUTER JOIN player USING (player_id)
	WHERE won = true AND dead = false AND ended_at IS NOT NULL
	ORDER BY playtime;`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[GameRecord])
}

func compilePlayerGameRecords(
	ctx context.Context, username string,
) ([]GameRecord, error) {
	rows, err := pg.db.Query(ctx, `
	SELECT username 
		, width
		, height
		, mine_count
		, "unique"
		, extract(epoch from (ended_at - started_at)) playtime
	FROM game_session
	LEFT OUTER JOIN player USING (player_id)
	WHERE username = $1 AND won = true AND dead = false AND ended_at IS NOT NULL
	ORDER BY playtime;`, username)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[GameRecord])
}
