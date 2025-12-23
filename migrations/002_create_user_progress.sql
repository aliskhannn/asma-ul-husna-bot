-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_progress
(
    user_id          bigint,
    name_number      smallint,

    -- SRS Core fields
    phase            VARCHAR(20)   DEFAULT 'new' CHECK (phase IN ('new', 'learning', 'mastered')),
    ease             NUMERIC(3, 2) DEFAULT 2.50 CHECK (ease BETWEEN 1.30 AND 2.50),
    streak           SMALLINT      DEFAULT 0 CHECK (streak >= 0),
    interval_days    INTEGER       DEFAULT 0 CHECK (interval_days >= 0),
    next_review_at   TIMESTAMPTZ   DEFAULT NULL,

    -- Tracking fields
    review_count     SMALLINT      DEFAULT 0 CHECK (review_count >= 0),
    correct_count    SMALLINT      DEFAULT 0 CHECK (correct_count >= 0),
    first_seen_at    TIMESTAMPTZ   DEFAULT NULL,
    last_reviewed_at TIMESTAMPTZ   DEFAULT NULL,
    introduced_at    TIMESTAMPTZ   DEFAULT NULL,
    created_at       timestamptz   DEFAULT NOW(),
    updated_at       timestamptz   DEFAULT NOW(),

    PRIMARY KEY (user_id, name_number),
    CONSTRAINT fk_user_progress_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_progress_srs_due
    ON user_progress (user_id, next_review_at)
    WHERE next_review_at IS NOT NULL;

CREATE INDEX idx_user_progress_phase
    ON user_progress (user_id, phase);

CREATE INDEX idx_user_progress_first_seen
    ON user_progress (user_id, first_seen_at)
    WHERE first_seen_at IS NOT NULL;

CREATE INDEX idx_user_progress_introduced_today
    ON user_progress (user_id, first_seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_progress_introduced_at
    ON user_progress (user_id, introduced_at);

COMMENT ON TABLE user_progress IS
    'Tracks user learning progress for each of the 99 Names of Allah using Spaced Repetition System (SRS)';

COMMENT ON COLUMN user_progress.phase IS
    'Learning phase: new (not studied), learning (in progress), mastered (fully learned)';

COMMENT ON COLUMN user_progress.ease IS
    'SRS ease factor (difficulty multiplier), range 1.30-2.50';

COMMENT ON COLUMN user_progress.streak IS
    'Number of consecutive correct answers';

COMMENT ON COLUMN user_progress.interval_days IS
    'Days until next review (SRS interval)';

COMMENT ON COLUMN user_progress.next_review_at IS
    'Scheduled time for next review';

COMMENT ON COLUMN user_progress.first_seen_at IS
    'When the name was first introduced to user (via reminder or manual view)';

COMMENT ON COLUMN user_progress.review_count IS
    'Total number of reviews (quiz attempts)';

COMMENT ON COLUMN user_progress.correct_count IS
    'Total number of correct answers';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_progress_introduced_today;
DROP INDEX IF EXISTS idx_user_progress_first_seen;
DROP INDEX IF EXISTS idx_user_progress_phase;
DROP INDEX IF EXISTS idx_user_progress_srs_due;
DROP INDEX IF EXISTS idx_user_progress_introduced_at;
DROP TABLE IF EXISTS user_progress;
-- +goose StatementEnd
