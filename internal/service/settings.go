package service

import (
	"context"
	"errors"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

type SettingsRepository interface {
	Create(ctx context.Context, userID int64) error
	GetByUserID(ctx context.Context, userID int64) (*entities.UserSettings, error)
	Update(ctx context.Context, settings *entities.UserSettings) error
	UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error
	UpdateQuizLength(ctx context.Context, userID int64, quizLength int) error
	UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error
	ToggleTransliteration(ctx context.Context, userID int64) error
	ToggleAudio(ctx context.Context, userID int64) error
}

type SettingsService struct {
	repository SettingsRepository
}

func NewSettingsService(repository SettingsRepository) *SettingsService {
	return &SettingsService{repository: repository}
}

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

func (s *SettingsService) Update(ctx context.Context, settings *entities.UserSettings) error {
	return s.repository.Update(ctx, settings)
}

func (s *SettingsService) UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error {
	return s.repository.UpdateNamesPerDay(ctx, userID, namesPerDay)
}

func (s *SettingsService) UpdateQuizLength(ctx context.Context, userID int64, quizLength int) error {
	return s.repository.UpdateQuizLength(ctx, userID, quizLength)
}

func (s *SettingsService) UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error {
	return s.repository.UpdateQuizMode(ctx, userID, quizMode)
}

func (s *SettingsService) ToggleTransliteration(ctx context.Context, userID int64) error {
	return s.repository.ToggleTransliteration(ctx, userID)
}

func (s *SettingsService) ToggleAudio(ctx context.Context, userID int64) error {
	return s.repository.ToggleAudio(ctx, userID)
}
