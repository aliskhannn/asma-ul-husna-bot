-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users
(
    id            bigint PRIMARY KEY,
    first_name    varchar(64) NOT NULL,
    last_name     varchar(64),
    username      varchar(32),
    language_code varchar(5),
    is_active     boolean     DEFAULT TRUE,
    created_at    timestamptz DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
