package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrRepositoryEmpty = errors.New("repository empty")
)

// namesWrapper is a helper struct for JSON unmarshaling.
type namesWrapper struct {
	Names []*entities.Name `json:"names"`
}

// NameRepository stores and manages the collection of Allah's names.
type NameRepository struct {
	names []*entities.Name
}

// NewNameRepository creates a new NameRepository from a JSON file.
// It reads the file, unmarshals the names, and ensures there are exactly 99 names.
func NewNameRepository(path string) (*NameRepository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper namesWrapper
	if err = json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	names := wrapper.Names

	if len(names) != 99 {
		return nil, fmt.Errorf("expected 99 names, got %d", len(names))
	}

	return &NameRepository{names: names}, nil
}

// GetByNumber returns the name with the specified number.
// If the number is out of range, it returns ErrNotFound.
func (r *NameRepository) GetByNumber(_ context.Context, number int) (*entities.Name, error) {
	if number < 1 || number > len(r.names) {
		return nil, ErrNotFound
	}
	return r.names[number-1], nil
}

// GetRandom returns a random name from the repository.
// If the repository is empty, it returns ErrRepositoryEmpty.
func (r *NameRepository) GetRandom(_ context.Context) (*entities.Name, error) {
	if len(r.names) == 0 {
		return nil, ErrRepositoryEmpty
	}
	idx := rand.Intn(len(r.names))
	return r.names[idx], nil
}

// GetAll returns all names stored in the repository.
func (r *NameRepository) GetAll(_ context.Context) ([]*entities.Name, error) {
	return r.names, nil
}
