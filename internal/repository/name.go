package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
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

func (r *NameRepository) GetNameByNumber(_ context.Context, number int) entities.Name {
	return r.names[number-1]
}

func (r *NameRepository) GetRandomName(_ context.Context) entities.Name {
	idx := rand.Intn(len(r.names))
	return r.names[idx]
}

func (r *NameRepository) GetAllNames(_ context.Context) []entities.Name {
	return r.names
}
