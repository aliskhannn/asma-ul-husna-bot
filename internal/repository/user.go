package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// UserRepository provides access to user data in the database.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository with the provided database pool.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// SaveUser inserts a new user into the database or updates an existing one.
// It sets IsActive and CreatedAt fields from the database.
func (r *UserRepository) SaveUser(ctx context.Context, user *entities.User) error {
	query := `
    INSERT INTO users (id, chat_id)
    VALUES ($1, $2)
    RETURNING is_active, created_at
    `
	err := r.db.QueryRow(ctx, query, user.ID, user.ChatID).Scan(&user.IsActive, &user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

// UserExists checks if a user with the given ID exists in the database.
func (r *UserRepository) UserExists(ctx context.Context, userID int64) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"

	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user existence: %w", err)
	}

	return exists, nil
}
