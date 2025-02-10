package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Player struct {
	PlayerId     int
	Username     string
	PasswordHash []byte
	CreatedAt    pgtype.Timestamptz
	UpdatedAt    pgtype.Timestamptz
}

type CreatePlayerParams struct {
	Username     string
	PasswordHash []byte
}

func (q *Queries) CreatePlayer_(ctx context.Context, params CreatePlayerParams) (*Player, error) {
	rows, _ := q.db.Query(
		ctx,
		"INSERT INTO player (username, password_hash) VALUES ($1, $2) RETURNING *",
		params.Username,
		params.PasswordHash,
	)
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Player])
}

func (q *Queries) FetchPlayer(ctx context.Context, username string) (*Player, error) {
	rows, _ := q.db.Query(
		ctx, "SELECT * FROM player WHERE username = $1", username,
	)
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Player])
}
