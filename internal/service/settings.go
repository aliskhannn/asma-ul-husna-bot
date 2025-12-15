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
