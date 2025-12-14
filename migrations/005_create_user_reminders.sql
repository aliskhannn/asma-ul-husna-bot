-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_reminders
(
    user_id bigint PRIMARY KEY,
    is_enabled boolean DEFAULT TRUE,
    frequency varchar(20) DEFAULT 'daily' CHECK (frequency IN ('daily', 'twice_daily', 'custom')),
    start_time_utc time NOT NULL DEFAULT '08:00:00', -- только время
    end_time_utc time NOT NULL DEFAULT '20:00:00',
    interval_hours smallint DEFAULT 24 CHECK (interval_hours > 0),
    last_sent_at timestamptz DEFAULT NULL,
    created_at timestamptz DEFAULT NOW(),
    updated_at timestamptz DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_reminders_enabled ON user_reminders(is_enabled, last_sent_at)
    WHERE is_enabled = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_reminders;

DROP INDEX idx_reminders_enabled;
-- +goose StatementEnd
