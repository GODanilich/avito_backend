-- +goose Up

CREATE TABLE pull_request_reviewers (
pull_request_id TEXT REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
user_id TEXT REFERENCES users(user_id),
PRIMARY KEY (pull_request_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS pull_request_reviewers;