-- name: GetTeamMembers :many
SELECT *
FROM users u
WHERE u.team_name = $1
ORDER BY u.user_id;


-- name: CreateTeam :exec
INSERT INTO teams (team_name) VALUES ($1);


-- name: GetTeam :one
SELECT t.team_name FROM teams t WHERE t.team_name = $1;