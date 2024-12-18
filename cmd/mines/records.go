package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/mines"
)

type GameRecord struct {
	GameSessionId string  `json:"session_id"`
	Username      *string `json:"username"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	MineCount     int     `json:"mine_count"`
	Unique        bool    `json:"unique"`
	Playtime      float64 `json:"playtime"`
}

type GameRecordFilters struct {
	username   *string
	gameParams *mines.GameParams
}

func (f GameRecordFilters) WhereClause() (string, pgx.NamedArgs) {
	args := pgx.NamedArgs{}
	whereClauses := []string{}
	if f.username != nil {
		args["username"] = &f.username
		whereClauses = append(whereClauses, "username = @username")
	}
	if f.gameParams != nil {
		args["width"] = &f.gameParams.Width
		args["height"] = &f.gameParams.Height
		args["mineCount"] = &f.gameParams.MineCount
		args["unique"] = &f.gameParams.Unique
		whereClauses = append(
			whereClauses,
			"width = @width",
			"height = @height",
			"mine_count = @mineCount",
			`"unique" = @unique`,
		)
	}

	if len(whereClauses) == 0 {
		return "", args
	}
	return strings.Join(whereClauses, " and "), args

}

type GameRecordsOption = func(*GameRecordFilters) error

func GameRecordsForPlayer(username string) GameRecordsOption {
	return func(f *GameRecordFilters) error {
		f.username = &username
		return nil
	}
}

func GameRecordsForGameParams(gameParams *mines.GameParams) GameRecordsOption {
	return func(f *GameRecordFilters) error {
		f.gameParams = gameParams
		return nil
	}
}

func getGameRecords(
	ctx context.Context, options ...GameRecordsOption,
) ([]GameRecord, error) {
	filters := &GameRecordFilters{}
	for _, op := range options {
		err := op(filters)
		if err != nil {
			return nil, err
		}
	}

	sql := `
	select 
		game_session_id
		, username 
		, width
		, height
		, mine_count
		, "unique"
		, (
			extract('epoch' from ended_at) - extract('epoch' from started_at)
		) * 1000 playtime
	from game_session
		left outer join player using (player_id)
	where 
		won = true 
		and dead = false 
		and ended_at is not null`

	whereClause, args := filters.WhereClause()
	if whereClause != "" {
		sql += " and " + whereClause
	}

	sql += " order by playtime"

	rows, err := pg.db.Query(ctx, sql, args)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[GameRecord])
}
