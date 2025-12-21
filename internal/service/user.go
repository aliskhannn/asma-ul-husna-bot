package service

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// UserService provides business logic for user management.
type UserService struct {
	repository UserRepository
}

// NewUserService creates a new UserService with the provided repository.
func NewUserService(repository UserRepository) *UserService {
	return &UserService{repository: repository}
}

// EnsureUser checks if a user exists and creates one if not.
// It does nothing if the user already exists.
func (s *UserService) EnsureUser(ctx context.Context, userID, chatID int64) error {
	user := entities.NewUser(userID, chatID)

	exists, err := s.repository.UserExists(ctx, user.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return s.repository.SaveUser(ctx, user)
}
