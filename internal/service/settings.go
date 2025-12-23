package service

import (
	"context"
	"errors"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

// SettingsService provides business logic for user settings management.
type SettingsService struct {
	repository SettingsRepository
}

// NewSettingsService creates a new SettingsService with the provided repository.
func NewSettingsService(repository SettingsRepository) *SettingsService {
	return &SettingsService{repository: repository}
}

// GetOrCreate retrieves user settings or creates default settings if they don't exist.
func (s *SettingsService) GetOrCreate(ctx context.Context, userID int64) (*entities.UserSettings, error) {
	settings, err := s.repository.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			// Create default settings.
			if err := s.repository.Create(ctx, userID); err != nil {
				return nil, err
			}
			// Retrieve newly created settings.
			return s.repository.GetByUserID(ctx, userID)
		}
		return nil, err
	}

	return settings, nil
}

// UpdateNamesPerDay updates the number of names to learn per day.
func (s *SettingsService) UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error {
	return s.repository.UpdateNamesPerDay(ctx, userID, namesPerDay)
}

// UpdateQuizMode updates the quiz mode setting.
func (s *SettingsService) UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error {
	return s.repository.UpdateQuizMode(ctx, userID, quizMode)
}

func (s *SettingsService) UpdateLearningMode(ctx context.Context, userID int64, learningMode string) error {
	return s.repository.UpdateLearningMode(ctx, userID, learningMode)
}
