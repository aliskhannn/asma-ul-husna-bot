package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres"
)

var ErrReminderNotFound = errors.New("reminder not found")

// ReminderRepository provides access to user reminder data in the database.
type ReminderRepository struct {
	db postgres.DBTX
}

// NewRemindersRepository creates a new ReminderRepository with the provided database pool.
func NewRemindersRepository(db postgres.DBTX) *ReminderRepository {
	return &ReminderRepository{db: db}
}

// GetByUserID retrieves reminder settings for a user.
func (r *ReminderRepository) GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error) {
	query := `
		SELECT user_id, is_enabled, interval_hours, start_time, end_time,
		       last_sent_at, next_send_at, last_kind, created_at, updated_at
		FROM user_reminders
		WHERE user_id = $1
	`

	var reminder entities.UserReminders
	var lastSent pgtype.Timestamptz
	var nextSend pgtype.Timestamptz
	var lastKind string

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&reminder.UserID,
		&reminder.IsEnabled,
		&reminder.IntervalHours,
		&reminder.StartTime,
		&reminder.EndTime,
		&lastSent,
		&nextSend,
		&lastKind,
		&reminder.CreatedAt,
		&reminder.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReminderNotFound
		}
		return nil, fmt.Errorf("get reminder: %w", err)
	}

	if lastSent.Valid {
		t := lastSent.Time
		reminder.LastSentAt = &t
	}
	if nextSend.Valid {
		t := nextSend.Time
		reminder.NextSendAt = &t
	}
	reminder.LastKind = entities.ReminderKind(lastKind)
	if reminder.LastKind == "" {
		reminder.LastKind = entities.ReminderKindNew
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
	if err != nil || timezone == "" {
		timezone = "UTC" // Fallback to UTC if not found
	}

	// Calculate next_send_at
	var nextSendAt time.Time
	if reminder.NextSendAt != nil {
		nextSendAt = *reminder.NextSendAt
	} else {
		nextSendAt = reminder.CalculateNextSendAt(timezone, time.Now())
	}

	query := `
		INSERT INTO user_reminders (
			user_id, is_enabled, interval_hours, start_time, end_time,
			last_sent_at, next_send_at, last_kind, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
			is_enabled = EXCLUDED.is_enabled,
			interval_hours = EXCLUDED.interval_hours,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			last_sent_at = EXCLUDED.last_sent_at,
			next_send_at = EXCLUDED.next_send_at,
			last_kind = EXCLUDED.last_kind,
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
		reminder.LastKind,
		reminder.CreatedAt,
		reminder.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	reminder.NextSendAt = &nextSendAt
	return nil
}

// GetDueReminder retrieves a single due reminder for a user.
func (r *ReminderRepository) GetDueReminder(ctx context.Context, userID int64) (*entities.ReminderWithUser, error) {
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
            ur.last_kind,
            COALESCE(us.timezone, 'UTC') as timezone
        FROM user_reminders ur
        INNER JOIN users u ON ur.user_id = u.id
        LEFT JOIN user_settings us ON ur.user_id = us.user_id
        WHERE ur.is_enabled = true
          AND u.is_active = true
          AND ur.user_id = $1
          AND (ur.next_send_at IS NULL OR ur.next_send_at <= $2)
        ORDER BY ur.next_send_at NULLS FIRST
        LIMIT 1
    `

	var rwu entities.ReminderWithUser
	var lastSent pgtype.Timestamptz
	var nextSend pgtype.Timestamptz
	var lastKind string

	now := time.Now()

	err := r.db.QueryRow(ctx, query, userID, now).Scan(
		&rwu.UserID,
		&rwu.ChatID,
		&rwu.IsEnabled,
		&rwu.IntervalHours,
		&rwu.StartTime,
		&rwu.EndTime,
		&lastSent,
		&nextSend,
		&lastKind,
		&rwu.Timezone,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReminderNotFound
		}
		return nil, fmt.Errorf("get due reminder: %w", err)
	}

	if lastSent.Valid {
		t := lastSent.Time
		rwu.LastSentAt = &t
	}
	if nextSend.Valid {
		t := nextSend.Time
		rwu.NextSendAt = &t
	}
	rwu.LastKind = entities.ReminderKind(lastKind)
	if rwu.LastKind == "" {
		rwu.LastKind = entities.ReminderKindNew
	}

	return &rwu, nil
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
			ur.last_kind,
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
		var lastSent pgtype.Timestamptz
		var nextSend pgtype.Timestamptz
		var lastKind string

		if err := rows.Scan(
			&rwu.UserID,
			&rwu.ChatID,
			&rwu.IsEnabled,
			&rwu.IntervalHours,
			&rwu.StartTime,
			&rwu.EndTime,
			&lastSent,
			&nextSend,
			&lastKind,
			&rwu.Timezone,
		); err != nil {
			return nil, fmt.Errorf("scan reminder: %w", err)
		}

		if lastSent.Valid {
			t := lastSent.Time
			rwu.LastSentAt = &t
		}
		if nextSend.Valid {
			t := nextSend.Time
			rwu.NextSendAt = &t
		}
		rwu.LastKind = entities.ReminderKind(lastKind)
		if rwu.LastKind == "" {
			rwu.LastKind = entities.ReminderKindNew
		}

		reminders = append(reminders, &rwu)
	}

	return reminders, rows.Err()
}

// UpdateAfterSend updates last_sent_at and next_send_at after sending a reminder.
func (r *ReminderRepository) UpdateAfterSend(ctx context.Context, userID int64, sentAt time.Time, nextSendAt time.Time, lastKind entities.ReminderKind) error {
	query := `
		UPDATE user_reminders
		SET last_sent_at = $1,
		    next_send_at = $2,
		    last_kind = $3,
		    updated_at = $4
		WHERE user_id = $5
	`

	result, err := r.db.Exec(ctx, query, sentAt, nextSendAt, lastKind, time.Now(), userID)
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

func (r *ReminderRepository) RescheduleNext(ctx context.Context, userID int64, nextSendAt time.Time) error {
	query := `
        UPDATE user_reminders
        SET next_send_at = $1,
            updated_at = $2
        WHERE user_id = $3
    `
	tag, err := r.db.Exec(ctx, query, nextSendAt, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("reschedule next: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReminderNotFound
	}
	return nil
}
