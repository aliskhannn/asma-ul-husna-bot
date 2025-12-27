package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres"
)

var ErrUserNotFound = errors.New("user not found")

// UserRepository provides access to user data in the database.
type UserRepository struct {
	db postgres.DBTX
}

// NewUserRepository creates a new UserRepository with the provided database pool.
func NewUserRepository(db postgres.DBTX) *UserRepository {
	return &UserRepository{db: db}
}

// Save inserts a new user or updates an existing one.
func (r *UserRepository) Save(ctx context.Context, user *entities.User) (bool, error) {
	query := `
		INSERT INTO users (id, chat_id, is_active, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			chat_id = EXCLUDED.chat_id,
			is_active = EXCLUDED.is_active
		RETURNING (xmax = 0) AS created
	`

	var created bool
	err := r.db.QueryRow(ctx, query, user.ID, user.ChatID, user.IsActive, user.CreatedAt).Scan(&created)
	if err != nil {
		return false, fmt.Errorf("save user: %w", err)
	}

	return created, nil
}

// Exists checks if a user with the given ID exists in the database.
func (r *UserRepository) Exists(ctx context.Context, userID int64) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"

	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user existence: %w", err)
	}

	return exists, nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, userID int64) (*entities.User, error) {
	query := `
		SELECT id, chat_id, is_active, created_at
		FROM users
		WHERE id = $1
	`

	var user entities.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.ChatID,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}
