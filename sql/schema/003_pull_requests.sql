-- +goose Up
DROP TYPE IF EXISTS pr_status;
CREATE TYPE pr_status AS ENUM ('OPEN', 'MERGED');

CREATE TABLE pull_requests (
pull_request_id UUID PRIMARY KEY,
pull_request_name TEXT NOT NULL,
author_id UUID NOT NULL REFERENCES users(user_id),
status pr_status NOT NULL DEFAULT 'OPEN',
created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
merged_at TIMESTAMP WITH TIME ZONE
);

-- +goose Down
DROP TABLE IF EXISTS pull_requests;