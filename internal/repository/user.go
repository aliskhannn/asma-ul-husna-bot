package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) SaveUser(ctx context.Context, user *entities.User) error {
	query := `
	INSERT INTO users (id, first_name, last_name, username, language_code)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING is_active, created_at
	
	`
	err := r.db.QueryRow(
		ctx, query, user.ID, user.FirstName, user.LastName, user.Username, user.LanguageCode,
	).Scan(&user.IsActive, &user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

func (r *UserRepository) UserExists(ctx context.Context, userID int64) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"

	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user existence: %w", err)
	}

	return exists, nil
}
