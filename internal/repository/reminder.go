package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

var ErrReminderNotFound = errors.New("reminder not found")

// ReminderRepository provides access to user reminder data in the database.
type ReminderRepository struct {
	db *pgxpool.Pool
}

// NewRemindersRepository creates a new ReminderRepository with the provided database pool.
func NewRemindersRepository(db *pgxpool.Pool) *ReminderRepository {
	return &ReminderRepository{db: db}
}

// GetByUserID retrieves reminder settings for a user.
func (r *ReminderRepository) GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error) {
	query := `
		SELECT user_id, is_enabled, interval_hours, start_time, end_time,
		       last_sent_at, next_send_at, created_at, updated_at
		FROM user_reminders
		WHERE user_id = $1
	`

	var reminder entities.UserReminders
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&reminder.UserID,
		&reminder.IsEnabled,
		&reminder.IntervalHours,
		&reminder.StartTime,
		&reminder.EndTime,
		&reminder.LastSentAt,
		&reminder.NextSendAt,
		&reminder.CreatedAt,
		&reminder.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReminderNotFound
		}
		return nil, fmt.Errorf("get reminder: %w", err)
	}

	return &reminder, nil
}

// Upsert creates or updates reminder settings.
func (r *ReminderRepository) Upsert(ctx context.Context, reminder *entities.UserReminders) error {
	// Get user's timezone
	var timezone string
	err := r.db.QueryRow(ctx,
		"SELECT timezone FROM user_settings WHERE user_id = $1",
		reminder.UserID,
	).Scan(&timezone)
	if err != nil {
		timezone = "UTC" // Fallback to UTC if not found
	}

	// Calculate next_send_at
	nextSendAt := reminder.CalculateNextSendAt(timezone, time.Now())

	query := `
		INSERT INTO user_reminders (
			user_id, is_enabled, interval_hours, start_time, end_time,
			last_sent_at, next_send_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			is_enabled = EXCLUDED.is_enabled,
			interval_hours = EXCLUDED.interval_hours,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			last_sent_at = EXCLUDED.last_sent_at,
			next_send_at = EXCLUDED.next_send_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.Exec(
		ctx,
		query,
		reminder.UserID,
		reminder.IsEnabled,
		reminder.IntervalHours,
		reminder.StartTime,
		reminder.EndTime,
		reminder.LastSentAt,
		nextSendAt,
		reminder.CreatedAt,
		reminder.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	return nil
}

// GetDueRemindersBatch retrieves reminders that are due to be sent (paginated).
func (r *ReminderRepository) GetDueRemindersBatch(ctx context.Context, now time.Time, limit, offset int) ([]*entities.ReminderWithUser, error) {
	query := `
		SELECT 
			ur.user_id,
			u.chat_id,
			ur.is_enabled,
			ur.interval_hours,
			ur.start_time,
			ur.end_time,
			ur.last_sent_at,
			ur.next_send_at,
			COALESCE(us.timezone, 'UTC') as timezone
		FROM user_reminders ur
		INNER JOIN users u ON ur.user_id = u.id
		LEFT JOIN user_settings us ON ur.user_id = us.user_id
		WHERE ur.is_enabled = true
			AND u.is_active = true
			AND (ur.next_send_at IS NULL OR ur.next_send_at <= $1)
		ORDER BY ur.next_send_at NULLS FIRST, ur.user_id
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, now, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get due reminders batch: %w", err)
	}
	defer rows.Close()

	var reminders []*entities.ReminderWithUser
	for rows.Next() {
		var rwu entities.ReminderWithUser
		err := rows.Scan(
			&rwu.UserID,
			&rwu.ChatID,
			&rwu.IsEnabled,
			&rwu.IntervalHours,
			&rwu.StartTime,
			&rwu.EndTime,
			&rwu.LastSentAt,
			&rwu.NextSendAt,
			&rwu.Timezone,
		)
		if err != nil {
			return nil, fmt.Errorf("scan reminder: %w", err)
		}
		reminders = append(reminders, &rwu)
	}

	return reminders, rows.Err()
}

// UpdateAfterSend updates last_sent_at and next_send_at after sending a reminder.
func (r *ReminderRepository) UpdateAfterSend(ctx context.Context, userID int64, sentAt time.Time, nextSendAt time.Time) error {
	query := `
		UPDATE user_reminders
		SET last_sent_at = $1,
		    next_send_at = $2,
		    updated_at = $3
		WHERE user_id = $4
	`

	result, err := r.db.Exec(ctx, query, sentAt, nextSendAt, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("update after send: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrReminderNotFound
	}

	return nil
}

// MarkAsSent updates the last sent timestamp for a reminder.
func (r *ReminderRepository) MarkAsSent(ctx context.Context, userID int64, sentAt time.Time) error {
	query := `
		UPDATE user_reminders
		SET last_sent_at = $1, updated_at = $2
		WHERE user_id = $3
	`

	result, err := r.db.Exec(ctx, query, sentAt, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("mark as sent: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrReminderNotFound
	}

	return nil
}
