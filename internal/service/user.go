package service

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) *UserService {
	return &UserService{repository: repository}
}

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
