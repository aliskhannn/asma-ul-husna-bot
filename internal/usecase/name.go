package usecase

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

type NameRepository interface {
	GetNameByNumber(_ context.Context, number int) (entities.Name, error)
	GetRandomName(_ context.Context) (entities.Name, error)
	GetAllNames(_ context.Context) ([]entities.Name, error)
}

type NameUseCase struct {
	repo NameRepository
}

func NewNameUseCase(repo NameRepository) *NameUseCase {
	return &NameUseCase{repo: repo}
}

func (uc *NameUseCase) GetNameByNumber(ctx context.Context, number int) (entities.Name, error) {
	return uc.repo.GetNameByNumber(ctx, number)
}

func (uc *NameUseCase) GetRandomName(ctx context.Context) (entities.Name, error) {
	return uc.repo.GetRandomName(ctx)
}

func (uc *NameUseCase) GetAllNames(ctx context.Context) ([]entities.Name, error) {
	return uc.repo.GetAllNames(ctx)
}
