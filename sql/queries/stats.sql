-- name: GetPRStats :many
SELECT status, COUNT(*) AS count
FROM pull_requests
GROUP BY status;

-- name: GetAssignmentStats :many
SELECT user_id, COUNT(*) AS count
FROM pull_request_reviewers
GROUP BY user_id;
