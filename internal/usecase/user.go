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

func (uc *UserUseCase) EnsureUser(ctx context.Context, user *entities.User) error {
	return uc.repo.EnsureUser(ctx, user)
}
