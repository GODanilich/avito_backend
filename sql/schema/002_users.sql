-- +goose Up

CREATE TABLE users (
user_id TEXT PRIMARY KEY,
username TEXT NOT NULL,
team_name TEXT REFERENCES teams(team_name) ON DELETE SET NULL,
is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_users_team_name ON users(team_name);
CREATE INDEX idx_users_team_active ON users(team_name, is_active);

-- +goose Down

DROP INDEX IF EXISTS idx_users_team_name;
DROP INDEX IF EXISTS idx_users_team_active;
DROP TABLE IF EXISTS users;