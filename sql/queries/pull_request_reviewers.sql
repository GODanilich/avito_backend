-- name: AddReviewer :exec
INSERT INTO pull_request_reviewers (pull_request_id, user_id)
VALUES ($1, $2) ON CONFLICT DO NOTHING;


-- name: GetPRReviewers :many
SELECT u.user_id FROM users u
JOIN pull_request_reviewers r ON u.user_id = r.user_id
WHERE r.pull_request_id = $1
ORDER BY u.user_id;


-- name: ReplaceReviewer :exec
DELETE FROM pull_request_reviewers
WHERE pull_request_id = $1 AND user_id = $2;
INSERT INTO pull_request_reviewers (pull_request_id, user_id)
VALUES ($1, $3);


-- name: GetPRsForReviewer :many
SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status
FROM pull_requests p
JOIN pull_request_reviewers r ON p.pull_request_id = r.pull_request_id
WHERE r.user_id = $1
ORDER BY p.created_at DESC;