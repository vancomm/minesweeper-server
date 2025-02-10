// custom query
package repository

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/mines"
)

type Highscore struct {
	GameSessionId string  `json:"game_session_id"`
	Username      *string `json:"username"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	MineCount     int     `json:"mine_count"`
	Unique        bool    `json:"unique"`
	PlaytimeMs    float64 `json:"playtime_ms"`
}

type HighscoreFilter struct {
	Username   *string
	GameParams *mines.GameParams
}

func (f HighscoreFilter) WhereClause() (string, pgx.NamedArgs) {
	clauses := make([]string, 0)
	args := pgx.NamedArgs{}
	if f.Username != nil {
		clauses = append(clauses, "username = @username")
		args["username"] = *f.Username
	}
	if f.GameParams != nil {
		clauses = append(
			clauses,
			"width = @width",
			"height = @height",
			"mine_count = @mineCount",
			`"unique" = @unique`,
		)
		args["width"] = f.GameParams.Width
		args["height"] = f.GameParams.Height
		args["mineCount"] = f.GameParams.MineCount
		args["unique"] = f.GameParams.Unique
	}
	return strings.Join(clauses, " AND "), args

}

func (q Queries) GetHighscores(
	ctx context.Context, filter HighscoreFilter,
) ([]Highscore, error) {
	query := `
	SELECT 
		game_session_id,
		username,
		width,
		height,
		mine_count,
		"unique",
		(
			extract('epoch' from ended_at) -
			extract('epoch' from started_at)
		) * 1000 playtime_ms
	FROM game_session
		LEFT OUTER JOIN player using (player_id)
	WHERE 
		won = true 
		AND dead = false 
		AND ended_at IS NOT NULL
	`

	whereClause, args := filter.WhereClause()
	if whereClause != "" {
		query += " AND " + whereClause
	}

	query += " ORDER BY playtime_ms;"

	rows, err := q.db.Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Highscore])
}
