-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_settings
(
    user_id bigint PRIMARY KEY,
    names_per_day smallint DEFAULT 1 CHECK (names_per_day BETWEEN 1 AND 20),
    quiz_length smallint DEFAULT 10 CHECK (quiz_length BETWEEN 5 AND 50),
    quiz_mode varchar(20) DEFAULT 'mixed' CHECK (quiz_mode IN ('new_only', 'review_only', 'mixed')),
    show_transliteration boolean DEFAULT TRUE,
    show_audio boolean DEFAULT TRUE,
    created_at timestamptz DEFAULT NOW(),
    updated_at timestamptz DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_settings;
-- +goose StatementEnd
