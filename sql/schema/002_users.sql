-- +goose Up

CREATE TABLE users (
user_id UUID PRIMARY KEY,
username TEXT NOT NULL,
team_name TEXT REFERENCES teams(team_name) ON DELETE SET NULL,
is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- +goose Down
DROP TABLE IF EXISTS users;