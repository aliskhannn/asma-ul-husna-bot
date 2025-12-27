-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_reminders
(
    user_id        bigint PRIMARY KEY,
    is_enabled     boolean             DEFAULT FALSE,
    interval_hours smallint            DEFAULT 1 CHECK (interval_hours IN (1, 2, 3, 4)),
    start_time     varchar(8) NOT NULL DEFAULT '08:00:00',
    end_time       varchar(8) NOT NULL DEFAULT '20:00:00',
    last_sent_at   timestamptz         DEFAULT NULL,
    next_send_at   timestamptz         DEFAULT NULL,
    last_kind      TEXT       NOT NULL DEFAULT 'new',
    created_at     timestamptz         DEFAULT NOW(),
    updated_at     timestamptz         DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_reminders_enabled ON user_reminders (is_enabled, last_sent_at)
    WHERE is_enabled = true;

CREATE INDEX IF NOT EXISTS idx_user_reminders_last_kind
    ON user_reminders (last_kind);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_reminders_enabled;
DROP INDEX idx_user_reminders_last_kind;

DROP TABLE IF EXISTS user_reminders;
-- +goose StatementEnd
