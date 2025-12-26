package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres"
)

var ErrSettingsNotFound = errors.New("settings not found")

// SettingsRepository provides access to user settings data in the database.
type SettingsRepository struct {
	db postgres.DBTX
}

// NewSettingsRepository creates a new SettingsRepository with the provided database pool.
func NewSettingsRepository(db postgres.DBTX) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Create creates default settings for a user.
func (r *SettingsRepository) Create(ctx context.Context, userID int64) error {
	query := `
		INSERT INTO user_settings (
			user_id, names_per_day, max_reviews_per_day, quiz_mode,
			learning_mode, language_code, timezone, created_at, updated_at
		) VALUES ($1, 1, 50, 'mixed', 'guided', 'ru', 'UTC', NOW(), NOW())
		ON CONFLICT (user_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("create settings: %w", err)
	}

	return nil
}

// GetByUserID retrieves settings for a user.
func (r *SettingsRepository) GetByUserID(ctx context.Context, userID int64) (*entities.UserSettings, error) {
	query := `
		SELECT user_id, names_per_day, max_reviews_per_day, quiz_mode,
		       learning_mode, language_code, timezone, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`

	var settings entities.UserSettings
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&settings.UserID,
		&settings.NamesPerDay,
		&settings.MaxReviewsPerDay,
		&settings.QuizMode,
		&settings.LearningMode,
		&settings.LanguageCode,
		&settings.Timezone,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSettingsNotFound
		}
		return nil, fmt.Errorf("get settings: %w", err)
	}

	return &settings, nil
}

func (r *SettingsRepository) UpsertDefaults(ctx context.Context, userID int64) error {
	query := `
		INSERT INTO user_settings (
			user_id, names_per_day, max_reviews_per_day, quiz_mode,
			learning_mode, language_code, timezone, created_at, updated_at
		) VALUES ($1, 1, 50, 'mixed', 'guided', 'ru', 'UTC', NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET names_per_day = EXCLUDED.names_per_day,
		    max_reviews_per_day = EXCLUDED.max_reviews_per_day,
		    quiz_mode = EXCLUDED.quiz_mode,
		    learning_mode = EXCLUDED.learning_mode,
		    language_code = EXCLUDED.language_code,
		    timezone = EXCLUDED.timezone,
		    updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("upsert default settings: %w", err)
	}
	return nil
}

// UpdateNamesPerDay updates the number of names to learn per day.
func (r *SettingsRepository) UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error {
	query := `
		UPDATE user_settings
		SET names_per_day = $1, updated_at = $2
		WHERE user_id = $3
	`

	result, err := r.db.Exec(ctx, query, namesPerDay, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("update names per day: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

// UpdateQuizMode updates the quiz mode setting.
func (r *SettingsRepository) UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error {
	query := `
		UPDATE user_settings
		SET quiz_mode = $1, updated_at = $2
		WHERE user_id = $3
	`

	result, err := r.db.Exec(ctx, query, quizMode, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("update quiz mode: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

// UpdateLearningMode updates the learning mode setting.
func (r *SettingsRepository) UpdateLearningMode(ctx context.Context, userID int64, learningMode string) error {
	query := `
		UPDATE user_settings
		SET learning_mode = $1, updated_at = $2
		WHERE user_id = $3
	`

	result, err := r.db.Exec(ctx, query, learningMode, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("update learning mode: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

// UpdateTimezone updates the user's timezone.
func (r *SettingsRepository) UpdateTimezone(ctx context.Context, userID int64, timezone string) error {
	query := `
		UPDATE user_settings
		SET timezone = $1, updated_at = $2
		WHERE user_id = $3
	`

	result, err := r.db.Exec(ctx, query, timezone, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("update timezone: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

// UpdateMaxReviewsPerDay updates the maximum reviews per day.
func (r *SettingsRepository) UpdateMaxReviewsPerDay(ctx context.Context, userID int64, maxReviews int) error {
	query := `
		UPDATE user_settings
		SET max_reviews_per_day = $1, updated_at = $2
		WHERE user_id = $3
	`

	result, err := r.db.Exec(ctx, query, maxReviews, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("update max reviews per day: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}
