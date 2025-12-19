package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

var ErrSettingsNotFound = errors.New("settings not found")

type SettingsRepository struct {
	db *pgxpool.Pool
}

func NewSettingsRepository(db *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Create creates default settings for a new user.
func (r *SettingsRepository) Create(ctx context.Context, userID int64) error {
	query := `
        INSERT INTO user_settings (user_id, names_per_day, quiz_mode, language_code, created_at, updated_at)
        VALUES ($1, 1, 'mixed', 'ru', NOW(), NOW())
        ON CONFLICT (user_id) DO NOTHING
    `

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("create settings: %w", err)
	}

	return nil
}

// GetByUserID retrieves settings by user ID.
// Returns ErrSettingsNotFound if settings don't exist.
func (r *SettingsRepository) GetByUserID(ctx context.Context, userID int64) (*entities.UserSettings, error) {
	query := `
        SELECT user_id, names_per_day, quiz_mode, language_code, created_at, updated_at
        FROM user_settings
        WHERE user_id = $1
    `

	var settings entities.UserSettings
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&settings.UserID,
		&settings.NamesPerDay,
		&settings.QuizMode,
		&settings.LanguageCode,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSettingsNotFound
		}
		return nil, fmt.Errorf("get settings by user id: %w", err)
	}

	return &settings, nil
}

// Update updates all settings fields.
func (r *SettingsRepository) Update(ctx context.Context, settings *entities.UserSettings) error {
	query := `
        UPDATE user_settings
        SET names_per_day = $2,
            quiz_mode = $3,
            language_code = $4,
            updated_at = NOW()
        WHERE user_id = $1
    `

	cmdTag, err := r.db.Exec(ctx, query,
		settings.UserID,
		settings.NamesPerDay,
		settings.QuizMode,
		settings.LanguageCode,
	)
	if err != nil {
		return fmt.Errorf("update settings: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

// UpdateNamesPerDay updates only the names_per_day field.
func (r *SettingsRepository) UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error {
	query := `
        UPDATE user_settings
        SET names_per_day = $2, updated_at = NOW()
        WHERE user_id = $1
    `

	cmdTag, err := r.db.Exec(ctx, query, userID, namesPerDay)
	if err != nil {
		return fmt.Errorf("update names per day: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

// UpdateQuizMode updates only the quiz_mode field.
func (r *SettingsRepository) UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error {
	query := `
        UPDATE user_settings
        SET quiz_mode = $2, updated_at = NOW()
        WHERE user_id = $1
    `

	cmdTag, err := r.db.Exec(ctx, query, userID, quizMode)
	if err != nil {
		return fmt.Errorf("update quiz mode: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

// Delete deletes settings for a user.
func (r *SettingsRepository) Delete(ctx context.Context, userID int64) error {
	query := `DELETE FROM user_settings WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete settings: %w", err)
	}

	return nil
}
