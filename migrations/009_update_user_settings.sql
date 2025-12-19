-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_settings
    DROP COLUMN IF EXISTS quiz_length,
    DROP COLUMN IF EXISTS show_transliteration,
    DROP COLUMN IF EXISTS show_audio;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_settings
    ADD COLUMN IF NOT EXISTS quiz_length smallint DEFAULT 10 CHECK (quiz_length BETWEEN 5 AND 50),
    ADD COLUMN IF NOT EXISTS show_transliteration boolean DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS show_audio boolean DEFAULT TRUE;
-- +goose StatementEnd
