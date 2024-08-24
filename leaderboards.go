package main

import "context"

type GameRecord struct {
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	MineCount int     `json:"mine_count"`
	Unique    bool    `json:"unique"`
	Playtime  float64 `json:"playtime"`
}

func compileGameRecords(ctx context.Context) ([]GameRecord, error) {
	records := []GameRecord{}
	rows, err := pg.db.Query(ctx, `
	SELECT width
		, height
		, mine_count
		, "unique"
		, extract(epoch from (ended_at - started_at)) playtime
	FROM game_session
	WHERE won = true AND dead = false AND ended_at IS NOT NULL
	ORDER BY playtime;`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var record GameRecord
		if err := rows.Scan(
			&record.Width,
			&record.Height,
			&record.MineCount,
			&record.Unique,
			&record.Playtime,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}
