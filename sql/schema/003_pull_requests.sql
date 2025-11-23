-- +goose Up
DROP TYPE IF EXISTS pr_status;
CREATE TYPE pr_status AS ENUM ('OPEN', 'MERGED');

CREATE TABLE pull_requests (
pull_request_id TEXT PRIMARY KEY,
pull_request_name TEXT NOT NULL,
author_id TEXT NOT NULL REFERENCES users(user_id),
status pr_status NOT NULL DEFAULT 'OPEN',
created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
merged_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_pull_requests_status ON pull_requests(status);
CREATE INDEX idx_pull_requests_created_at ON pull_requests(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_pull_requests_status;
DROP INDEX IF EXISTS idx_pull_requests_created_at;
DROP TABLE IF EXISTS pull_requests;