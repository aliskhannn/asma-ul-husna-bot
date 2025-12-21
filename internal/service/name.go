package service

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// NameService provides business logic for working with Allah's names.
type NameService struct {
	repository NameRepository
}

// NewNameService creates a new NameService with the provided repository.
func NewNameService(repository NameRepository) *NameService {
	return &NameService{repository: repository}
}

// GetByNumber retrieves a name by its number from the repository.
func (s *NameService) GetByNumber(ctx context.Context, number int) (*entities.Name, error) {
	return s.repository.GetByNumber(ctx, number)
}

// GetRandom retrieves a random name from the repository.
func (s *NameService) GetRandom(ctx context.Context) (*entities.Name, error) {
	return s.repository.GetRandom(ctx)
}

// GetAll retrieves all names from the repository.
func (s *NameService) GetAll(ctx context.Context) ([]*entities.Name, error) {
	return s.repository.GetAll(ctx)
}
