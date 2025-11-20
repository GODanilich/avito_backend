-- name: UpsertUser :exec
INSERT INTO users (user_id, username, team_name, is_active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET username = EXCLUDED.username, team_name = EXCLUDED.team_name, is_active = EXCLUDED.is_active;

-- name: GetUserById :one
SELECT u.user_id, u.username, u.team_name, u.is_active
FROM users u
WHERE u.user_id = $1;

-- name: SetUserActive :one
UPDATE users
SET is_active = $2
WHERE user_id = $1
RETURNING user_id, username, team_name, is_active;