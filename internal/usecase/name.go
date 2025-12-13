package usecase

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

type NameRepository interface {
	GetNameByNumber(_ context.Context, number int) entities.Name
	GetRandomName(_ context.Context) entities.Name
	GetAllNames(_ context.Context) []entities.Name
}

type NameUseCase struct {
	repo NameRepository
}

func NewNameUseCase(repo NameRepository) *NameUseCase {
	return &NameUseCase{repo: repo}
}

func (uc *NameUseCase) GetNameByNumber(ctx context.Context, number int) entities.Name {
	return uc.repo.GetNameByNumber(ctx, number)
}

func (uc *NameUseCase) GetRandomName(ctx context.Context) entities.Name {
	return uc.repo.GetRandomName(ctx)
}

func (uc *NameUseCase) GetAllNames(ctx context.Context) []entities.Name {
	return uc.repo.GetAllNames(ctx)
}
