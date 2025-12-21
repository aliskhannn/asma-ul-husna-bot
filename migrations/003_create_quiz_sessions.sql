-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS quiz_sessions
(
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    current_question_num smallint NOT NULL,
    correct_answers smallint DEFAULT 0,
    total_questions smallint NOT NULL,
    quiz_mode varchar(20), -- "new", "review", "mixed"
    session_status varchar(15) DEFAULT 'active',-- "active", "completed", "abandoned"
    started_at timestamptz DEFAULT NOW(),
    completed_at timestamptz DEFAULT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_session_status CHECK ( session_status IN ('active', 'completed', 'abandoned') )
);

CREATE INDEX idx_quiz_sessions_user_active ON quiz_sessions(user_id, session_status)
    WHERE session_status = 'active';
CREATE INDEX idx_quiz_sessions_completed ON quiz_sessions(user_id, completed_at)
    WHERE session_status = 'completed';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_quiz_sessions_user_active;
DROP INDEX idx_quiz_sessions_completed;

DROP TABLE IF EXISTS quiz_sessions CASCADE;
-- +goose StatementEnd
