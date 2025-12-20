-- +goose Up
-- +goose StatementBegin
-- migrations/006_create_user_daily_name.up.sql
CREATE TABLE IF NOT EXISTS user_daily_name
(
    user_id     BIGINT    NOT NULL,
    date_utc    DATE      NOT NULL,
    name_number SMALLINT  NOT NULL,
    slot_index  SMALLINT  NOT NULL DEFAULT 0,    -- ДОБАВИЛИ
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, date_utc, slot_index), -- ИЗМЕНИЛИ
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_daily_name_date ON user_daily_name (date_utc);
CREATE INDEX idx_user_daily_name_user_date ON user_daily_name (user_id, date_utc);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_daily_name;
-- +goose StatementEnd
