package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

var (
	ErrNameNotFound  = errors.New("name not found")
	ErrInvalidNumber = errors.New("invalid name number")
)

// NameRepository provides access to the 99 Names of Allah.
// This implementation uses an in-memory dataset, but you could load from DB or JSON.
type NameRepository struct {
	names []*entities.Name
}

// NewNameRepository creates a new NameRepository with the 99 Names.
func NewNameRepository(path string) (*NameRepository, error) {
	names, err := get99Names(path)
	if err != nil {
		return nil, err
	}

	return &NameRepository{
		names: names,
	}, nil
}

// GetByNumber returns the name with the specified number.
// If the number is out of range, it returns ErrNotFound.
// GetByNumber retrieves a name by its number (1-99).
func (r *NameRepository) GetByNumber(number int) (*entities.Name, error) {
	if number < 1 || number > 99 {
		return nil, ErrInvalidNumber
	}

	for _, name := range r.names {
		if name.Number == number {
			return name, nil
		}
	}

	return nil, ErrNameNotFound
}

// GetRandom retrieves a random name.
func (r *NameRepository) GetRandom() (*entities.Name, error) {
	if len(r.names) == 0 {
		return nil, ErrNameNotFound
	}

	idx := rand.Intn(len(r.names))
	return r.names[idx], nil
}

// GetAll retrieves all 99 names.
func (r *NameRepository) GetAll() ([]*entities.Name, error) {
	return r.names, nil
}

// GetByNumbers retrieves multiple names by their numbers.
func (r *NameRepository) GetByNumbers(numbers []int) ([]entities.Name, error) {
	result := make([]entities.Name, 0, len(numbers))

	for _, num := range numbers {
		name, err := r.GetByNumber(num)
		if err != nil {
			return nil, fmt.Errorf("get name %d: %w", num, err)
		}
		result = append(result, *name)
	}

	return result, nil
}

func get99Names(path string) ([]*entities.Name, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Names []*entities.Name `json:"names"`
	}
	if err = json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal names JSON: %w", err)
	}

	if len(wrapper.Names) != 99 {
		return nil, fmt.Errorf("expected 99 names, got %d", len(wrapper.Names))
	}

	return wrapper.Names, nil
}
