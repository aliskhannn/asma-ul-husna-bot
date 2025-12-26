package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres"
)

// DailyNameRepository manages daily introduced names.
type DailyNameRepository struct {
	db postgres.DBTX
}

// NewDailyNameRepository creates a new DailyNameRepository.
func NewDailyNameRepository(db postgres.DBTX) *DailyNameRepository {
	return &DailyNameRepository{db: db}
}

// GetTodayNames retrieves names introduced today.
func (r *DailyNameRepository) GetTodayNames(ctx context.Context, userID int64) ([]int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	query := `
		SELECT name_number
		FROM user_daily_name
		WHERE user_id = $1 AND date_utc = $2
		ORDER BY slot_index
	`

	rows, err := r.db.Query(ctx, query, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get today names: %w", err)
	}
	defer rows.Close()

	var names []int
	for rows.Next() {
		var nameNumber int
		if err := rows.Scan(&nameNumber); err != nil {
			return nil, fmt.Errorf("scan name number: %w", err)
		}
		names = append(names, nameNumber)
	}

	return names, rows.Err()
}

// GetTodayNamesCount returns the count of names introduced today.
func (r *DailyNameRepository) GetTodayNamesCount(ctx context.Context, userID int64) (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	query := `
		SELECT COUNT(*)
		FROM user_daily_name
		WHERE user_id = $1 AND date_utc = $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, userID, today).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get today names count: %w", err)
	}

	return count, nil
}

// HasUnfinishedDays returns true if there are previous days with names not learned yet.
func (r *DailyNameRepository) HasUnfinishedDays(ctx context.Context, userID int64) (bool, error) {
	query := `
		SELECT EXISTS (
  			SELECT 1
  				FROM user_daily_name udn
					LEFT JOIN user_progress up
  						ON up.user_id = udn.user_id AND up.name_number = udn.name_number
  				WHERE udn.user_id = $1
    				AND udn.date_utc < (NOW() AT TIME ZONE 'UTC')::date
    				AND COALESCE(up.streak, 0) < 7
		)
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("has unfinished days: %w", err)
	}

	return exists, nil
}

func (r *DailyNameRepository) GetOldestUnfinishedName(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT udn.name_number
		FROM user_daily_name udn
			LEFT JOIN user_progress up
  				ON up.user_id = udn.user_id AND up.name_number = udn.name_number
		WHERE udn.user_id = $1
 	 		AND udn.date_utc < (NOW() AT TIME ZONE 'UTC')::date
  			AND COALESCE(up.streak, 0) < 7
		ORDER BY udn.date_utc, udn.slot_index
		LIMIT 1
`
	var name int
	if err := r.db.QueryRow(ctx, query, userID).Scan(&name); err != nil {
		return 0, fmt.Errorf("get oldest unfinished name: %w", err)
	}
	return name, nil
}

// AddTodayName adds a name to today's introduced names.
func (r *DailyNameRepository) AddTodayName(ctx context.Context, userID int64, nameNumber int) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Get next slot index
	var slotIndex int
	query := `
		SELECT COALESCE(MAX(slot_index), -1) + 1
		FROM user_daily_name
		WHERE user_id = $1 AND date_utc = $2
	`
	err := r.db.QueryRow(ctx, query, userID, today).Scan(&slotIndex)
	if err != nil {
		return fmt.Errorf("get next slot index: %w", err)
	}

	// Insert
	insertQuery := `
		INSERT INTO user_daily_name (user_id, date_utc, name_number, slot_index)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, date_utc, slot_index) DO NOTHING
	`

	_, err = r.db.Exec(ctx, insertQuery, userID, today, nameNumber, slotIndex)
	if err != nil {
		return fmt.Errorf("add today name: %w", err)
	}

	return nil
}

// RemoveTodayName removes a name from today's list (when it moves to learning/mastered).
func (r *DailyNameRepository) RemoveTodayName(ctx context.Context, userID int64, nameNumber int) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	query := `
		DELETE FROM user_daily_name
		WHERE user_id = $1 AND date_utc = $2 AND name_number = $3
	`

	_, err := r.db.Exec(ctx, query, userID, today, nameNumber)
	if err != nil {
		return fmt.Errorf("remove today name: %w", err)
	}

	return nil
}
