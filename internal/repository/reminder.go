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

var ErrReminderNotFound = errors.New("reminder record not found")

type ReminderRepository struct {
	db *pgxpool.Pool
}

func NewRemindersRepository(db *pgxpool.Pool) *ReminderRepository {
	return &ReminderRepository{db: db}
}

func (r *ReminderRepository) GetDueReminders(ctx context.Context) ([]*entities.ReminderWithUser, error) {
	query := `
        SELECT
    		r.user_id,
   			r.is_enabled,
    		r.interval_hours,
    		r.start_time_utc,
    		r.end_time_utc,
    		r.last_sent_at,
    		r.created_at,
    		r.updated_at,
    		u.id          AS user_id,
    		u.chat_id     AS chat_id
		FROM user_reminders r
		JOIN users u ON u.id = r.user_id
		WHERE r.is_enabled = true
  			AND (
      			r.last_sent_at IS NULL
      			OR r.last_sent_at < (
          			NOW() AT TIME ZONE 'UTC'
          			- (r.interval_hours || ' hours')::interval
      			)
  			)
  			AND EXTRACT(HOUR FROM NOW() AT TIME ZONE 'UTC') BETWEEN
      			EXTRACT(HOUR FROM r.start_time_utc) AND
      			EXTRACT(HOUR FROM r.end_time_utc)
		ORDER BY r.user_id;
    `

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get due reminders: %w", err)
	}
	defer rows.Close()

	var res []*entities.ReminderWithUser
	for rows.Next() {
		var rwu entities.ReminderWithUser
		if err := rows.Scan(
			&rwu.UserID,
			&rwu.IsEnabled,
			&rwu.IntervalHours,
			&rwu.StartTimeUTC,
			&rwu.EndTimeUTC,
			&rwu.LastSentAt,
			&rwu.CreatedAt,
			&rwu.UpdatedAt,
			&rwu.UserID,
			&rwu.ChatID,
		); err != nil {
			return nil, fmt.Errorf("scan reminder: %w", err)
		}
		res = append(res, &rwu)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}

	return res, nil
}

func (r *ReminderRepository) MarkAsSent(ctx context.Context, userID int64, sentAt time.Time) error {
	query := `
		UPDATE user_reminders
		SET last_sent_at = $2,
		    updated_at = NOW()
		WHERE user_id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, userID, sentAt)
	if err != nil {
		return fmt.Errorf("mark as sent: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("mark as sent: %s", ErrReminderNotFound)
	}

	return nil
}

func (r *ReminderRepository) GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error) {
	query := `
        SELECT user_id, is_enabled, interval_hours, start_time_utc, 
               end_time_utc, last_sent_at, created_at, updated_at
        FROM user_reminders
        WHERE user_id = $1
    `
	var rem entities.UserReminders
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&rem.UserID,
		&rem.IsEnabled,
		&rem.IntervalHours,
		&rem.StartTimeUTC,
		&rem.EndTimeUTC,
		&rem.LastSentAt,
		&rem.CreatedAt,
		&rem.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get reminder by user id: %w", err)
	}

	return &rem, nil
}

func (r *ReminderRepository) Upsert(ctx context.Context, rem *entities.UserReminders) error {
	query := `
        INSERT INTO user_reminders
            (user_id, is_enabled, interval_hours, start_time_utc, end_time_utc, last_sent_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id) DO UPDATE SET
            is_enabled     = EXCLUDED.is_enabled,
            interval_hours = EXCLUDED.interval_hours,
            start_time_utc = EXCLUDED.start_time_utc,
            end_time_utc   = EXCLUDED.end_time_utc,
            last_sent_at   = EXCLUDED.last_sent_at,
            updated_at     = NOW()
    `
	_, err := r.db.Exec(ctx, query,
		rem.UserID,
		rem.IsEnabled,
		rem.IntervalHours,
		rem.StartTimeUTC,
		rem.EndTimeUTC,
		rem.LastSentAt,
	)
	if err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}
	return nil
}
