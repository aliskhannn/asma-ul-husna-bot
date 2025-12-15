package service

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type UserRepository interface {
	SaveUser(ctx context.Context, user *entities.User) error
	UserExists(ctx context.Context, userID int64) (bool, error)
}

type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) *UserService {
	return &UserService{repository: repository}
}

func (s *UserService) EnsureUser(
	ctx context.Context,
	userID int64,
	firstName, lastName string,
	username string,
	languageCode string,
) error {
	user := entities.NewUser(userID, firstName, &lastName, &username, &languageCode)

	exists, err := s.repository.UserExists(ctx, user.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return s.repository.SaveUser(ctx, user)
}
