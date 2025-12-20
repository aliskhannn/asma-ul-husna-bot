-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_settings
(
    user_id bigint PRIMARY KEY,
    names_per_day smallint DEFAULT 1 CHECK (names_per_day BETWEEN 1 AND 20),
    max_reviews_per_day SMALLINT DEFAULT 50 CHECK (max_reviews_per_day BETWEEN 10 AND 200),
    quiz_mode varchar(20) DEFAULT 'mixed' CHECK (quiz_mode IN ('new', 'review', 'mixed', 'daily')),
    language_code varchar(5) NOT NULL DEFAULT 'ru',
    created_at timestamptz DEFAULT NOW(),
    updated_at timestamptz DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_settings;
-- +goose StatementEnd
