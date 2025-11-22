-- name: CreatePR :exec
INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
VALUES ($1,$2,$3,'OPEN', NOW());

-- name: GetPR :one
SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
FROM pull_requests
WHERE pull_request_id = $1;


-- name: SetPRMerged :one
UPDATE pull_requests
SET status='MERGED', merged_at = now()
WHERE pull_request_id = $1
RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at;

-- name: GetActiveReviewersForTeam :many
SELECT user_id
FROM users
WHERE team_name = $1
AND is_active = TRUE
AND user_id <> $2;