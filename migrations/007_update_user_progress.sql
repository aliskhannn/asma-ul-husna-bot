-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_progress
    ADD COLUMN IF NOT EXISTS phase VARCHAR(20) DEFAULT 'new' CHECK ( phase IN ('new', 'learning', 'mastered') ),
    ADD COLUMN IF NOT EXISTS ease NUMERIC(3,2) DEFAULT 2.50,
    ADD COLUMN IF NOT EXISTS streak SMALLINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS interval_days INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS next_review_at TIMESTAMPTZ DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_user_progress_srs_due
    ON user_progress(user_id, next_review_at)
    WHERE next_review_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_progress_phase
    ON user_progress(user_id, phase);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_progress
    DROP COLUMN IF EXISTS phase,
    DROP COLUMN IF EXISTS ease,
    DROP COLUMN IF EXISTS streak,
    DROP COLUMN IF EXISTS interval_days,
    DROP COLUMN IF EXISTS next_review_at;

DROP INDEX IF EXISTS idx_user_progress_srs_due;
DROP INDEX IF EXISTS idx_user_progress_phase;
-- +goose StatementEnd
