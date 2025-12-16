package service

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type NameRepository interface {
	GetByNumber(_ context.Context, number int) (*entities.Name, error)
	GetRandom(_ context.Context) (*entities.Name, error)
	GetAll(_ context.Context) ([]*entities.Name, error)
}

type NameService struct {
	repository NameRepository
}

func NewNameService(repository NameRepository) *NameService {
	return &NameService{repository: repository}
}

func (s *NameService) GetByNumber(ctx context.Context, number int) (*entities.Name, error) {
	return s.repository.GetByNumber(ctx, number)
}

func (s *NameService) GetRandom(ctx context.Context) (*entities.Name, error) {
	return s.repository.GetRandom(ctx)
}

func (s *NameService) GetAll(ctx context.Context) ([]*entities.Name, error) {
	return s.repository.GetAll(ctx)
}
