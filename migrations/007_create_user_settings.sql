-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_settings
(
    user_id             bigint PRIMARY KEY,
    names_per_day       smallint             DEFAULT 1 CHECK (names_per_day BETWEEN 1 AND 20),
    max_reviews_per_day smallint             DEFAULT 50 CHECK (max_reviews_per_day BETWEEN 10 AND 200),
    quiz_mode           varchar(20)          DEFAULT 'mixed' CHECK (quiz_mode IN ('new', 'review', 'mixed')),
    learning_mode       varchar(20)          DEFAULT 'guided' CHECK (learning_mode IN ('guided', 'free')),
    language_code       varchar(5)  NOT NULL DEFAULT 'ru',
    timezone            varchar(50) NOT NULL DEFAULT 'UTC',
    created_at          timestamptz          DEFAULT NOW(),
    updated_at          timestamptz          DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_settings_learning_mode
    ON user_settings (learning_mode);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_settings_learning_mode;
DROP TABLE IF EXISTS user_settings;
-- +goose StatementEnd
