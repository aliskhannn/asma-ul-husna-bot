-- +goose Up
-- +goose StatementBegin

ALTER TABLE user_reminders
    ADD COLUMN IF NOT EXISTS last_kind TEXT NOT NULL DEFAULT 'new';

CREATE INDEX IF NOT EXISTS idx_user_reminders_last_kind
    ON user_reminders (last_kind);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_user_reminders_last_kind;

ALTER TABLE user_reminders
    DROP COLUMN IF EXISTS last_kind;

-- +goose StatementEnd
