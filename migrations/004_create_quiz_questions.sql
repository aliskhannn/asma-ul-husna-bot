-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS quiz_questions
(
    id             bigserial PRIMARY KEY,
    session_id     bigint      NOT NULL,
    question_order smallint    NOT NULL,
    name_number    smallint    NOT NULL,
    question_type  varchar(20) NOT NULL,
    correct_answer text        NOT NULL,
    options        TEXT[]      DEFAULT '{}',
    correct_index  INTEGER     DEFAULT 0,
    created_at     timestamptz DEFAULT NOW(),
    updated_at     timestamptz DEFAULT NOW(),

    FOREIGN KEY (session_id) REFERENCES quiz_sessions (id) ON DELETE CASCADE,
    UNIQUE (session_id, question_order)
);

CREATE INDEX idx_quiz_questions_session ON quiz_questions (session_id, question_order);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_quiz_questions_session;

DROP TABLE IF EXISTS quiz_questions CASCADE;
-- +goose StatementEnd
