-- +goose Up
-- +goose StatementBegin
-- 1) Привести существующие значения к новым (иначе новый CHECK может не примениться)
UPDATE user_settings
SET quiz_mode = CASE quiz_mode
                    WHEN 'review_only' THEN 'review'
                    WHEN 'new_only'    THEN 'new'
                    ELSE quiz_mode
    END;

-- 2) Переустановить CHECK constraint
ALTER TABLE user_settings
    DROP CONSTRAINT IF EXISTS user_settings_quiz_mode_check,
    ADD CONSTRAINT user_settings_quiz_mode_check
        CHECK (quiz_mode IN ('new', 'review', 'mixed', 'daily'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Откат значений назад
UPDATE user_settings
SET quiz_mode = CASE quiz_mode
                    WHEN 'review' THEN 'review_only'
                    WHEN 'new'    THEN 'new_only'
                    ELSE quiz_mode
    END;

ALTER TABLE user_settings
    DROP CONSTRAINT IF EXISTS user_settings_quiz_mode_check,
    ADD CONSTRAINT user_settings_quiz_mode_check
        CHECK (quiz_mode IN ('new_only', 'review_only', 'mixed', 'daily'));
-- +goose StatementEnd
