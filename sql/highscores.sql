-- name: GetAllHighScores :many

SELECT 
    game_session_id
    , username 
    , width
    , height
    , mine_count
    , "unique"
    , (
        extract('epoch' from ended_at) - extract('epoch' from started_at)
    ) * 1000 playtime_ms
FROM game_session
    LEFT OUTER JOIN player using (player_id)
WHERE 
    won = true 
    AND dead = false 
    AND ended_at IS NOT NULL;

-- name: GetHighScoresForUsername :many

SELECT 
    game_session_id
    , username 
    , width
    , height
    , mine_count
    , "unique"
    , (
        extract('epoch' from ended_at) - extract('epoch' from started_at)
    ) * 1000 playtime_ms
FROM game_session
    LEFT OUTER JOIN player using (player_id)
WHERE 
    won = true 
    AND dead = false 
    AND ended_at IS NOT NULL
    AND username = $1;

-- name: GetHighScoresForGameParams :many

SELECT 
    game_session_id
    , username 
    , width
    , height
    , mine_count
    , "unique"
    , (
        extract('epoch' from ended_at) - extract('epoch' from started_at)
    ) * 1000 playtime_ms
FROM game_session
    LEFT OUTER JOIN player using (player_id)
WHERE 
    won = true 
    AND dead = false 
    AND ended_at IS NOT NULL
    AND width = $1
    AND height = $2
    AND mine_count = $3
    AND "unique" = $4;

-- name: GetHighScoresForUsernameAndGameParams :many

SELECT 
    game_session_id
    , username 
    , width
    , height
    , mine_count
    , "unique"
    , (
        extract('epoch' from ended_at) - extract('epoch' from started_at)
    ) * 1000 playtime_ms
FROM game_session
    LEFT OUTER JOIN player using (player_id)
WHERE 
    won = true 
    AND dead = false 
    AND ended_at IS NOT NULL
    AND username = $1
    AND width = $2
    AND height = $3
    AND mine_count = $4
    AND "unique" = $5;