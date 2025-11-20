-- name: InsertPR :exec
INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
VALUES ($1,$2,$3,'OPEN');

-- name: GetPR :one
SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
FROM pull_requests
WHERE pull_request_id = $1;


-- name: SetPRMerged :one
UPDATE pull_requests
SET status='MERGED', merged_at = now()
WHERE pull_request_id = $1
RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at;