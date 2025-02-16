package repository

import (
	"bytes"
	"context"
	"encoding/gob"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/vancomm/minesweeper-server/internal/mines"
)

type GameSession struct {
	GameSessionId int
	PlayerId      *int
	Width         int
	Height        int
	MineCount     int
	Unique        bool
	Dead          bool
	Won           bool
	StartedAt     pgtype.Timestamptz
	EndedAt       pgtype.Timestamptz
	State         []byte
	CreatedAt     pgtype.Timestamptz
	UpdatedAt     pgtype.Timestamptz
}

type CreateGameSessionParams struct {
	PlayerId *int
}

func (p CreateGameSessionParams) UpdateArgs(args *pgx.NamedArgs) *pgx.NamedArgs {
	if p.PlayerId != nil {
		(*args)["player_id"] = *p.PlayerId
	}
	return args
}

func (q Queries) CreateGameSession(
	ctx context.Context, state *mines.GameState, params CreateGameSessionParams,
) (*GameSession, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return nil, err
	}

	args := pgx.NamedArgs{
		"width":      state.Width,
		"height":     state.Height,
		"mine_count": state.MineCount,
		"unique":     state.Unique,
		"dead":       state.Dead,
		"won":        state.Won,
		"state":      buf.Bytes(),
	}
	params.UpdateArgs(&args)

	rows, _ := q.db.Query(
		ctx,
		`INSERT INTO game_session (
			player_id, width, height, mine_count, "unique", dead, won, state
		) 
		VALUES (
			@player_id, @width, @height, @mine_count, @unique, @dead, @won, @state
		) 
		RETURNING *;`,
		args,
	)
	return pgx.CollectExactlyOneRow(
		rows, pgx.RowToAddrOfStructByName[GameSession],
	)
}

func (q Queries) FetchGameSession(ctx context.Context, gameSessionId int) (*GameSession, error) {
	rows, _ := q.db.Query(
		ctx,
		"SELECT * FROM game_session WHERE game_session_id = $1",
		gameSessionId,
	)
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[GameSession])
}

type UpdateGameSessionParams struct {
	Dead    *bool
	Won     *bool
	EndedAt *time.Time
	State   *[]byte
}

func (p UpdateGameSessionParams) SetClause() (string, map[string]any) {
	parts := make([]string, 0)
	args := make(map[string]any)

	if p.Dead != nil {
		parts = append(parts, "dead = @dead")
		args["dead"] = *p.Dead
	}
	if p.Won != nil {
		parts = append(parts, "won = @won")
		args["won"] = *p.Won
	}
	if p.EndedAt != nil {
		parts = append(parts, "ended_at = @ended_at")
		args["ended_at"] = *p.EndedAt
	}
	if p.State != nil {
		parts = append(parts, "state = @state")
		args["state"] = *p.State
	}

	return strings.Join(parts, ", "), args
}

func (q Queries) UpdateGameSession(
	ctx context.Context, gameSessionId int, params UpdateGameSessionParams,
) (*GameSession, error) {
	setClause, args := params.SetClause()
	args["game_session_id"] = gameSessionId
	rows, _ := q.db.Query(
		ctx,
		"UPDATE game_session SET "+setClause+" WHERE game_session_id = @game_session_id RETURNING *",
		pgx.NamedArgs(args),
	)
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[GameSession])
}
