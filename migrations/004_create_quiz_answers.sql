-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS quiz_answers
(
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    session_id bigint NOT NULL,
    name_number smallint NOT NULL,
    user_answer text,
    correct_answer text,
    question_type varchar(20), -- "translation", "transliteration", "meaning"
    is_correct boolean NOT NULL,
    answered_at timestamptz DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES quiz_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_quiz_answers_user_name ON quiz_answers(user_id, name_number, is_correct);
CREATE INDEX idx_quiz_answers_session ON quiz_answers(session_id);
CREATE INDEX idx_quiz_answers_date ON quiz_answers(user_id, answered_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS quiz_answers;

DROP INDEX idx_quiz_answers_user_name;
DROP INDEX idx_quiz_answers_session;
DROP INDEX idx_quiz_answers_date;
-- +goose StatementEnd
