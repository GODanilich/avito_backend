-- +goose Up

CREATE TABLE teams (
team_name TEXT PRIMARY KEY
);

-- +goose Down
DROP TABLE IF EXISTS teams;