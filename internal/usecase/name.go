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
	nameRepository NameRepository
}

func NewNameUseCase(nameRepository NameRepository) *NameUseCase {
	return &NameUseCase{nameRepository: nameRepository}
}

func (r *NameUseCase) GetNameByNumber(ctx context.Context, number int) (entities.Name, error) {
	return r.nameRepository.GetNameByNumber(ctx, number)
}

func (r *NameUseCase) GetRandomName(ctx context.Context) (entities.Name, error) {
	return r.nameRepository.GetRandomName(ctx)
}

func (r *NameUseCase) GetAllNames(ctx context.Context) ([]entities.Name, error) {
	return r.nameRepository.GetAllNames(ctx)
}
