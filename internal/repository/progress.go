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

type ProgressRepository struct {
	db *pgxpool.Pool
}

func NewProgressRepository(db *pgxpool.Pool) *ProgressRepository {
	return &ProgressRepository{db: db}
}

// ProgressStats contains user progress statistics for /progress command.
type ProgressStats struct {
	TotalViewed    int        // Total names viewed (has progress record)
	Learned        int        // Names marked as learned
	InProgress     int        // Names viewed but not learned
	NotStarted     int        // Names never viewed (99 - TotalViewed)
	Accuracy       float64    // Average correct_count for learned names
	LastActivityAt *time.Time // Last review timestamp
}

// Upsert creates or updates a progress record.
func (r *ProgressRepository) Upsert(ctx context.Context, progress *entities.UserProgress) error {
	query := `
		INSERT INTO user_progress (user_id, name_number, is_learned, last_reviewed_at, correct_count)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, name_number)
		DO UPDATE SET
			is_learned = excluded.is_learned,
			last_reviewed_at = excluded.last_reviewed_at,
			correct_count = excluded.correct_count
	`

	_, err := r.db.Exec(
		ctx, query,
		progress.UserID,
		progress.NameNumber,
		progress.IsLearned,
		progress.LastReviewedAt,
		progress.CorrectCount,
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
		SELECT user_id, name_number, is_learned, last_reviewed_at, correct_count
		FROM user_progress 
		WHERE user_id = $1 AND name_number = $2
	`

	var progress entities.UserProgress
	err := r.db.QueryRow(
		ctx, query, userID, nameNumber,
	).Scan(
		&progress.UserID,
		&progress.NameNumber,
		&progress.IsLearned,
		&progress.LastReviewedAt,
		&progress.CorrectCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProgressNotFound
		}

		return nil, fmt.Errorf("get: %w", err)
	}

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

// DeleteByUserID deletes all progress records for a given user.
func (r *ProgressRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	query := `DELETE FROM user_progress WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete by user id: %w", err)
	}

	return nil
}

// MarkAsViewed creates a progress record when user views a name via it's number.
// Does nothing if record already exists.
func (r *ProgressRepository) MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error {
	query := `
		INSERT INTO user_progress (user_id, name_number, is_learned, last_reviewed_at, correct_count)
		VALUES ($1, $2, false, NULL, $5)
		ON CONFLICT (user_id, name_number) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, nameNumber)
	if err != nil {
		return fmt.Errorf("mark as viewed: %w", err)
	}

	return nil
}

// GetViewedCount returns the count of names the user has viewed.
func (r *ProgressRepository) GetViewedCount(ctx context.Context, userID int64) (int, error) {
	query := "SELECT COUNT(*) FROM user_progress WHERE user_id = $1"

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get viewed count: %w", err)
	}

	return count, nil
}

// MarkAsLearned marks a name as learned (sets is_learned = true).
func (r *ProgressRepository) MarkAsLearned(ctx context.Context, userID int64, nameNumber int) error {
	query := `
		INSERT INTO user_progress (user_id, name_number, is_learned, last_reviewed_at, correct_count)
		VALUES ($1, $2, false, NULL, $5)
		ON CONFLICT (user_id, name_number)
		DO UPDATE SET is_learned = true
	`

	_, err := r.db.Exec(ctx, query, userID, nameNumber)
	if err != nil {
		return fmt.Errorf("mark as learned: %w", err)
	}

	return nil
}

// IsLearned checks if a name is marked as learned.
func (r *ProgressRepository) IsLearned(ctx context.Context, userID int64, nameNumber int) (bool, error) {
	query := `
		SELECT is_learned
		FROM user_progress
		WHERE user_id = $1 AND name_number = $2
	`

	var isLearned bool
	err := r.db.QueryRow(ctx, query, userID, nameNumber).Scan(&isLearned)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // not viewed yet = not learned
		}

		return false, fmt.Errorf("is learned: %w", err)
	}

	return isLearned, nil
}

// GetLearnedNames returns a list of name numbers that are marked as learned.
func (r *ProgressRepository) GetLearnedNames(ctx context.Context, userID int64) ([]int, error) {
	query := `
		SELECT name_number
		FROM user_progress
		WHERE user_id = $1 AND is_learned = true
		ORDER BY name_number
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get learned names: %w", err)
	}
	defer rows.Close()

	var nameNumbers []int
	for rows.Next() {
		var num int
		if err = rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("scan learned name: %w", err)
		}
		nameNumbers = append(nameNumbers, num)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learned names: %w", err)
	}

	return nameNumbers, nil
}

// CountLearned returns the count of learned names.
func (r *ProgressRepository) CountLearned(ctx context.Context, userID int64) (int, error) {
	query := `
        SELECT COUNT(*) 
        FROM user_progress 
        WHERE user_id = $1 AND is_learned = true
    `

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count learned: %w", err)
	}

	return count, nil
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

// GetNamesToReview returns names that need review based on spaced repetition algorithm.
// Returns learned names that either:
// - Have never been reviewed (last_reviewed_at IS NULL)
// - Haven't been reviewed in 3+ days
func (r *ProgressRepository) GetNamesToReview(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
		SELECT name_number
		FROM user_progress
		WHERE user_id = $1
			AND is_learned = true
			AND (last_reviewed_at IS NULL OR last_reviewed_at < NOW() - INTERVAL '3 days')
		ORDER BY last_reviewed_at NULLS FIRST
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get names to review: %w", err)
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

// IncrementCorrectCount increments the correct answer counter.
func (r *ProgressRepository) IncrementCorrectCount(ctx context.Context, userID int64, nameNumber int) error {
	query := `
        UPDATE user_progress 
        SET correct_count = correct_count + 1
        WHERE user_id = $1 AND name_number = $2
    `

	cmdTag, err := r.db.Exec(ctx, query, userID, nameNumber)
	if err != nil {
		return fmt.Errorf("increment correct count: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrProgressNotFound
	}

	return nil
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

// GetInProgressNames returns names that are viewed but not yet learned.
func (r *ProgressRepository) GetInProgressNames(ctx context.Context, userID int64) ([]int, error) {
	query := `
        SELECT name_number 
        FROM user_progress 
        WHERE user_id = $1 AND is_learned = false
        ORDER BY name_number ASC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get in progress names: %w", err)
	}
	defer rows.Close()

	nameNumbers := make([]int, 0)
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("scan in progress name: %w", err)
		}
		nameNumbers = append(nameNumbers, num)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate in progress names: %w", err)
	}

	return nameNumbers, nil
}

// CountInProgress returns the count of names in progress (viewed but not learned).
func (r *ProgressRepository) CountInProgress(ctx context.Context, userID int64) (int, error) {
	query := `
        SELECT COUNT(*) 
        FROM user_progress 
        WHERE user_id = $1 AND is_learned = false
    `

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count in progress: %w", err)
	}

	return count, nil
}

// GetStats returns comprehensive statistics for /progress command.
func (r *ProgressRepository) GetStats(ctx context.Context, userID int64) (*ProgressStats, error) {
	query := `
        SELECT 
            COUNT(*) as total_viewed,
            COUNT(*) FILTER (WHERE is_learned = true) as learned,
            COUNT(*) FILTER (WHERE is_learned = false) as in_progress,
            COALESCE(AVG(correct_count) FILTER (WHERE is_learned = true), 0) as avg_correct,
            MAX(last_reviewed_at) as last_activity
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
	)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	stats.NotStarted = 99 - stats.TotalViewed
	stats.LastActivityAt = lastActivity

	return &stats, nil
}

// GetLastActivityDate returns the date of last review activity.
func (r *ProgressRepository) GetLastActivityDate(ctx context.Context, userID int64) (*time.Time, error) {
	query := `
        SELECT MAX(last_reviewed_at) 
        FROM user_progress 
        WHERE user_id = $1 AND last_reviewed_at IS NOT NULL
    `

	var lastActivity *time.Time
	err := r.db.QueryRow(ctx, query, userID).Scan(&lastActivity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get last activity date: %w", err)
	}

	return lastActivity, nil
}
