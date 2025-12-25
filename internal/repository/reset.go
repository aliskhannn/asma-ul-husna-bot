package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ResetRepository struct {
	db *pgxpool.Pool
}

func NewResetRepository(db *pgxpool.Pool) *ResetRepository {
	return &ResetRepository{
		db: db,
	}
}

func (s *ResetRepository) ResetUser(ctx context.Context, userID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Порядок зависит от FK.
	if _, err := tx.Exec(ctx, `DELETE FROM quiz_sessions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_daily_name WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_progress WHERE user_id = $1`, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE user_reminders
		SET last_sent_at = NULL, next_send_at = NULL, last_kind = 'new', updated_at = NOW()
		WHERE user_id = $1
	`, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
