-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_settings
    ADD COLUMN IF NOT EXISTS max_reviews_per_day SMALLINT DEFAULT 50 CHECK (max_reviews_per_day BETWEEN 10 AND 200);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_settings
    DROP COLUMN IF EXISTS max_reviews_per_day;
-- +goose StatementEnd
