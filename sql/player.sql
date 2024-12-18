-- name: GetPlayer :one
SELECT * FROM player
WHERE username = $1;

-- name: CreatePlayer :one
INSERT INTO player (username, password_hash)
VALUES ($1, $2)
RETURNING *;