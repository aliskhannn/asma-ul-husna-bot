-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_progress
(
    user_id          bigint,
    name_number      smallint,
    is_learned       boolean   DEFAULT FALSE,
    last_reviewed_at timestamptz DEFAULT NULL,
    correct_count    smallint  DEFAULT 0,

    PRIMARY KEY (user_id, name_number),
    CONSTRAINT fk_user_progress_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_progress_review ON user_progress(user_id, last_reviewed_at) WHERE is_learned = true;
CREATE INDEX idx_user_progress_learning ON user_progress(user_id, is_learned);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_progress;

DROP INDEX idx_user_progress_review;
DROP INDEX idx_user_progress_learning;
-- +goose StatementEnd
