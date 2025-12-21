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

var ErrProgressNotFound = errors.New("progress not found")

// ProgressRepository provides access to user progress data in the database.
type ProgressRepository struct {
	db *pgxpool.Pool
}

// NewProgressRepository creates a new ProgressRepository with the provided database pool.
func NewProgressRepository(db *pgxpool.Pool) *ProgressRepository {
	return &ProgressRepository{db: db}
}

// Upsert creates or updates a progress record for a user and name.
func (r *ProgressRepository) Upsert(ctx context.Context, progress *entities.UserProgress) error {
	query := `
		INSERT INTO user_progress (
			user_id, name_number, is_learned, last_reviewed_at, correct_count,
			phase, ease, streak, interval_days, next_review_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, name_number)
		DO UPDATE SET
			is_learned = excluded.is_learned,
			last_reviewed_at = excluded.last_reviewed_at,
			correct_count = excluded.correct_count,
			phase = excluded.phase,
			ease = excluded.ease,
			streak = excluded.streak,
			interval_days = excluded.interval_days,
			next_review_at = excluded.next_review_at
	`

	_, err := r.db.Exec(
		ctx, query,
		progress.UserID,
		progress.NameNumber,
		progress.IsLearned,
		progress.LastReviewedAt,
		progress.CorrectCount,
		progress.Phase,
		progress.Ease,
		progress.Streak,
		progress.IntervalDays,
		progress.NextReviewAt,
	)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}

	return nil
}

// Get retrieves a single progress record by userID and nameNumber.
// Returns ErrProgressNotFound if the record doesn't exist.
func (r *ProgressRepository) Get(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error) {
	query := `
		SELECT user_id, name_number, is_learned, last_reviewed_at, correct_count,
		       COALESCE(phase, 'new') as phase,
		       COALESCE(ease, 2.5) as ease,
		       COALESCE(streak, 0) as streak,
		       COALESCE(interval_days, 0) as interval_days,
		       next_review_at
		FROM user_progress
		WHERE user_id = $1 AND name_number = $2
	`

	var progress entities.UserProgress
	var phase string
	err := r.db.QueryRow(
		ctx, query, userID, nameNumber,
	).Scan(
		&progress.UserID,
		&progress.NameNumber,
		&progress.IsLearned,
		&progress.LastReviewedAt,
		&progress.CorrectCount,
		&phase,
		&progress.Ease,
		&progress.Streak,
		&progress.IntervalDays,
		&progress.NextReviewAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProgressNotFound
		}
		return nil, fmt.Errorf("get: %w", err)
	}

	progress.Phase = entities.Phase(phase)
	return &progress, nil
}

// GetByUserID retrieves all progress records for a given user.
func (r *ProgressRepository) GetByUserID(ctx context.Context, userID int64) ([]*entities.UserProgress, error) {
	query := `
		SELECT user_id, name_number, is_learned, last_reviewed_at, correct_count
		FROM user_progress 
		WHERE user_id = $1
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get by user id: %w", err)
	}
	defer rows.Close()

	var progress []*entities.UserProgress
	for rows.Next() {
		var p entities.UserProgress
		err = rows.Scan(
			&p.UserID,
			&p.NameNumber,
			&p.IsLearned,
			&p.LastReviewedAt,
			&p.CorrectCount,
		)
		if err != nil {
			return nil, fmt.Errorf("get by user id: %w", err)
		}
		progress = append(progress, &p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("get by user id: %w", err)
	}

	return progress, nil
}

// MarkAsViewed creates a progress record when user views a name via it's number.
// Does nothing if record already exists.
func (r *ProgressRepository) MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error {
	query := `
		INSERT INTO user_progress (user_id, name_number, is_learned, last_reviewed_at, correct_count)
		VALUES ($1, $2, false, NULL, 0)
		ON CONFLICT (user_id, name_number) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, nameNumber)
	if err != nil {
		return fmt.Errorf("mark as viewed: %w", err)
	}

	return nil
}

// RecordReview updates progress after a quiz answer.
// Updates last_reviewed_at and increments correct_count if answer is correct.
func (r *ProgressRepository) RecordReview(ctx context.Context, userID int64, nameNumber int, isCorrect bool, reviewedAt time.Time) error {
	query := `
		INSERT INTO user_progress (user_id, name_number, is_learned, last_reviewed_at, correct_count)
		VALUES ($1, $2, false, $3, CASE WHEN $4 THEN 1 ELSE 0 END)
		ON CONFLICT (user_id, name_number)
		DO UPDATE SET
			last_reviewed_at = excluded.last_reviewed_at,
			correct_count = CASE
				WHEN $4 THEN user_progress.correct_count + 1
				ELSE user_progress.correct_count
			END
	`

	_, err := r.db.Exec(ctx, query, userID, nameNumber, reviewedAt, isCorrect)
	if err != nil {
		return fmt.Errorf("record review: %w", err)
	}

	return nil
}

// GetNamesDueForReview retrieves names that need to be reviewed right now (SRS).
func (r *ProgressRepository) GetNamesDueForReview(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
		SELECT name_number
		FROM user_progress
		WHERE user_id = $1
		  AND next_review_at IS NOT NULL
		  AND next_review_at <= NOW()
		ORDER BY next_review_at
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get names due for review: %w", err)
	}
	defer rows.Close()

	nameNumbers := make([]int, 0, limit)
	for rows.Next() {
		var num int
		if err = rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("scan review name: %w", err)
		}
		nameNumbers = append(nameNumbers, num)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review names: %w", err)
	}

	return nameNumbers, nil
}

// GetNewNames returns name numbers that haven't been learned yet.
// Used for generating "new only" quizzes.
func (r *ProgressRepository) GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
        WITH all_names AS (
            SELECT generate_series(1, 99) AS name_number
        )
        SELECT an.name_number
        FROM all_names an
        LEFT JOIN user_progress up ON an.name_number = up.name_number AND up.user_id = $1
        WHERE up.name_number IS NULL OR up.is_learned = false
        ORDER BY an.name_number ASC
        LIMIT $2
    `

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get new names: %w", err)
	}
	defer rows.Close()

	nameNumbers := make([]int, 0, limit)
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("scan new name: %w", err)
		}
		nameNumbers = append(nameNumbers, num)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate new names: %w", err)
	}

	return nameNumbers, nil
}

// ProgressStats contains user progress statistics for /progress command.
type ProgressStats struct {
	TotalViewed    int
	Learned        int
	InProgress     int
	NotStarted     int
	Accuracy       float64
	LastActivityAt *time.Time

	// SRS поля
	NewCount      int     // phase = 'new'
	LearningCount int     // phase = 'learning'
	MasteredCount int     // phase = 'mastered'
	DueToday      int     // next_review_at <= NOW()
	AverageEase   float64 // средний ease
}

// GetStats returns comprehensive statistics for /progress command.
func (r *ProgressRepository) GetStats(ctx context.Context, userID int64) (*ProgressStats, error) {
	query := `
		SELECT
			COUNT(*) as total_viewed,
			COUNT(*) FILTER (WHERE is_learned = true) as learned,
			COUNT(*) FILTER (WHERE is_learned = false) as in_progress,
			COALESCE(AVG(correct_count) FILTER (WHERE is_learned = true), 0) as avg_correct,
			MAX(last_reviewed_at) as last_activity,
			COUNT(*) FILTER (WHERE phase = 'new') as new_count,
			COUNT(*) FILTER (WHERE phase = 'learning') as learning_count,
			COUNT(*) FILTER (WHERE phase = 'mastered') as mastered_count,
			COUNT(*) FILTER (WHERE next_review_at IS NOT NULL AND next_review_at <= NOW()) as due_today,
			COALESCE(AVG(ease), 2.5) as avg_ease
		FROM user_progress
		WHERE user_id = $1
	`

	var stats ProgressStats
	var lastActivity *time.Time
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.TotalViewed,
		&stats.Learned,
		&stats.InProgress,
		&stats.Accuracy,
		&lastActivity,
		&stats.NewCount,
		&stats.LearningCount,
		&stats.MasteredCount,
		&stats.DueToday,
		&stats.AverageEase,
	)

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	stats.NotStarted = 99 - stats.TotalViewed
	stats.LastActivityAt = lastActivity
	return &stats, nil
}

// GetNextDueName retrieves the next name due for review for a user.
// Returns 0 if no names are due.
func (r *ProgressRepository) GetNextDueName(ctx context.Context, userID int64) (int, error) {
	query := `
        SELECT name_number
        FROM user_progress
        WHERE user_id = $1 
          AND next_review_at IS NOT NULL 
          AND next_review_at <= NOW()
        ORDER BY next_review_at
        LIMIT 1
    `

	var nameNumber int
	err := r.db.QueryRow(ctx, query, userID).Scan(&nameNumber)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil // нет due имён
		}
		return 0, fmt.Errorf("get next due name: %w", err)
	}

	return nameNumber, nil
}

// GetOrCreateDailyName retrieves or creates a daily name for a user.
// If no daily name exists for the date, assigns a new one.
func (r *ProgressRepository) GetOrCreateDailyName(
	ctx context.Context,
	userID int64,
	dateUTC time.Time,
	namesPerDay int,
) (int, error) {
	if namesPerDay < 1 {
		namesPerDay = 1
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queryGet := `
        SELECT name_number 
        FROM user_daily_name 
        WHERE user_id = $1 AND date_utc = $2
        ORDER BY slot_index
        FOR UPDATE
    `

	rows, err := tx.Query(ctx, queryGet, userID, dateUTC.Format("2006-01-02"))
	if err != nil {
		return 0, fmt.Errorf("check daily names: %w", err)
	}

	var existingNumbers []int
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan daily name: %w", err)
		}
		existingNumbers = append(existingNumbers, num)
	}
	rows.Close()

	if len(existingNumbers) > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit transaction: %w", err)
		}
		return existingNumbers[0], nil
	}

	queryNew := `
        SELECT n.number
        FROM generate_series(1, 99) AS n(number)
        WHERE NOT EXISTS (
            SELECT 1 FROM user_progress up 
            WHERE up.user_id = $1 AND up.name_number = n.number
        )
        ORDER BY n.number
        LIMIT $2
    `

	rows, err = tx.Query(ctx, queryNew, userID, namesPerDay)
	if err != nil {
		return 0, fmt.Errorf("get new names for daily: %w", err)
	}

	var newNumbers []int
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan new name: %w", err)
		}
		newNumbers = append(newNumbers, num)
	}
	rows.Close()

	if len(newNumbers) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit transaction: %w", err)
		}
		return 0, nil
	}

	queryInsert := `
        INSERT INTO user_daily_name (user_id, date_utc, name_number, slot_index, created_at)
        VALUES ($1, $2, $3, $4, NOW())
        ON CONFLICT (user_id, date_utc, slot_index) DO NOTHING
    `

	for i, num := range newNumbers {
		_, err = tx.Exec(ctx, queryInsert, userID, dateUTC.Format("2006-01-02"), num, i)
		if err != nil {
			return 0, fmt.Errorf("save daily name: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return newNumbers[0], nil
}

// GetNextDailyName retrieves the next daily name for review for a user on a specific date.
// Returns 0 if no daily names are available.
func (r *ProgressRepository) GetNextDailyName(
	ctx context.Context,
	userID int64,
	dateUTC time.Time,
) (int, error) {
	query := `
        SELECT udn.name_number
        FROM user_daily_name udn
        LEFT JOIN user_progress up 
            ON up.user_id = udn.user_id 
            AND up.name_number = udn.name_number
        WHERE udn.user_id = $1 
          AND udn.date_utc = $2
          AND (up.last_reviewed_at IS NULL OR up.last_reviewed_at < udn.created_at)
        ORDER BY udn.slot_index
        LIMIT 1
    `

	var nameNumber int
	err := r.db.QueryRow(ctx, query, userID, dateUTC.Format("2006-01-02")).Scan(&nameNumber)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get next daily name: %w", err)
	}

	return nameNumber, nil
}

// GetRandomLearnedName retrieves a random learned name for a user.
// Returns 0 if no names are learned.
func (r *ProgressRepository) GetRandomLearnedName(ctx context.Context, userID int64) (int, error) {
	query := `
        SELECT name_number
        FROM user_progress
        WHERE user_id = $1
        ORDER BY RANDOM()
        LIMIT 1
    `

	var nameNumber int
	err := r.db.QueryRow(ctx, query, userID).Scan(&nameNumber)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get random learned name: %w", err)
	}

	return nameNumber, nil
}
