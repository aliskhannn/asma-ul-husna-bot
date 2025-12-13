package usecase

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

type UserRepository interface {
	EnsureUser(ctx context.Context, user *entities.User) error
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
	return uc.repo.EnsureUser(ctx, user)
}
