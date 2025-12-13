package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrRepositoryEmpty = errors.New("repository empty")
)

type namesWrapper struct {
	Names []entities.Name `json:"names"`
}

type NameRepository struct {
	names []entities.Name
}

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

func (r *NameRepository) GetNameByNumber(_ context.Context, number int) (entities.Name, error) {
	if number < 1 || number > len(r.names) {
		return entities.Name{}, ErrNotFound
	}
	return r.names[number-1], nil
}

func (r *NameRepository) GetRandomName(_ context.Context) (entities.Name, error) {
	if len(r.names) == 0 {
		return entities.Name{}, ErrRepositoryEmpty
	}
	idx := rand.Intn(len(r.names))
	return r.names[idx], nil
}

func (r *NameRepository) GetAllNames(_ context.Context) ([]entities.Name, error) {
	return r.names, nil
}
