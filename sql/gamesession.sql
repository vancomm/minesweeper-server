-- name: GetSession :one

SELECT * FROM game_session 
WHERE game_session_id = $1 LIMIT 1;

-- name: CreatePlayerSession :one

INSERT INTO game_session (
    player_id, width, height, mine_count, "unique", dead, won, "state"
) 
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING game_session_id, "state", started_at;

-- name: CreateAnonymousSession :one

INSERT INTO game_session (
    width, height, mine_count, "unique", dead, won, "state"
) 
VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING game_session_id, "state", started_at;

-- name: UpdateSession :exec

UPDATE game_session
SET dead = $1,
    won = $2,
    ended_at = $3,
    "state" = $4
WHERE game_session_id = $5;