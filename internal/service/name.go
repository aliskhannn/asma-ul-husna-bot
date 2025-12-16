package service

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type NameRepository interface {
	GetNameByNumber(_ context.Context, number int) (entities.Name, error)
	GetRandomName(_ context.Context) (entities.Name, error)
	GetAllNames(_ context.Context) ([]entities.Name, error)
}

type NameService struct {
	repository NameRepository
}

func NewNameService(repository NameRepository) *NameService {
	return &NameService{repository: repository}
}

func (s *NameService) GetByNumber(ctx context.Context, number int) (entities.Name, error) {
	return s.repository.GetNameByNumber(ctx, number)
}

func (s *NameService) GetRandom(ctx context.Context) (entities.Name, error) {
	return s.repository.GetRandomName(ctx)
}

func (s *NameService) GetAll(ctx context.Context) ([]entities.Name, error) {
	return s.repository.GetAllNames(ctx)
}
