package usecase

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

type UserRepository interface {
	SaveUser(ctx context.Context, user *entities.User) error
	UserExists(ctx context.Context, userID int64) (bool, error)
}

type UserUseCase struct {
	repo UserRepository
}

func NewUserUseCase(repo UserRepository) *UserUseCase {
	return &UserUseCase{repo: repo}
}

func (uc *UserUseCase) EnsureUser(
	ctx context.Context,
	userID int64,
	firstName, lastName string,
	username string,
	languageCode string,
) error {
	user := entities.NewUser(userID, firstName, lastName, username, languageCode)

	exists, err := uc.repo.UserExists(ctx, user.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	
	return uc.repo.SaveUser(ctx, user)
}
