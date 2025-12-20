-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_progress
(
    user_id          bigint,
    name_number      smallint,
    is_learned       boolean   DEFAULT FALSE,
    last_reviewed_at timestamptz DEFAULT NULL,
    correct_count    smallint  DEFAULT 0,
    phase            VARCHAR(20) DEFAULT 'new' CHECK ( phase IN ('new', 'learning', 'mastered') ),
    ease             NUMERIC(3,2) DEFAULT 2.50,
    streak           SMALLINT DEFAULT 0,
    interval_days    INTEGER DEFAULT 0,
    next_review_at   TIMESTAMPTZ DEFAULT NULL,

    PRIMARY KEY (user_id, name_number),
    CONSTRAINT fk_user_progress_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_progress_review ON user_progress(user_id, last_reviewed_at) WHERE is_learned = true;
CREATE INDEX idx_user_progress_learning ON user_progress(user_id, is_learned);
CREATE INDEX idx_user_progress_srs_due ON user_progress(user_id, next_review_at) WHERE next_review_at IS NOT NULL;
CREATE INDEX idx_user_progress_phase ON user_progress(user_id, phase);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_progress_srs_due;
DROP INDEX IF EXISTS idx_user_progress_phase;
DROP INDEX idx_user_progress_review;
DROP INDEX idx_user_progress_learning;

DROP TABLE IF EXISTS user_progress;
-- +goose StatementEnd
