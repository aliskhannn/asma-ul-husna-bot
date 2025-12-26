package repository

import (
	"context"
	"fmt"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres"
)

type ResetRepository struct {
	db postgres.DBTX
}

func NewResetRepository(db postgres.DBTX) *ResetRepository {
	return &ResetRepository{db: db}
}

func (s *ResetRepository) ResetUser(ctx context.Context, userID int64) error {
	if _, err := s.db.Exec(ctx, `DELETE FROM quiz_sessions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete quiz_sessions: %w", err)
	}
	if _, err := s.db.Exec(ctx, `DELETE FROM user_daily_name WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete user_daily_name: %w", err)
	}
	if _, err := s.db.Exec(ctx, `DELETE FROM user_progress WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete user_progress: %w", err)
	}

	return nil
}
