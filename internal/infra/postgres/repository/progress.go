package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres"
)

var ErrProgressNotFound = errors.New("progress not found")

// ProgressRepository provides access to user progress data in the database.
type ProgressRepository struct {
	db postgres.DBTX
}

// NewProgressRepository creates a new ProgressRepository with the provided database pool.
func NewProgressRepository(db postgres.DBTX) *ProgressRepository {
	return &ProgressRepository{db: db}
}

// Upsert creates or updates a progress record within a transaction.
func (r *ProgressRepository) Upsert(ctx context.Context, progress *entities.UserProgress) error {
	query := `
		INSERT INTO user_progress (
			user_id, name_number, phase, ease, streak, interval_days,
			next_review_at, review_count, correct_count, first_seen_at, last_reviewed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, name_number) DO UPDATE SET
			phase = EXCLUDED.phase,
			ease = EXCLUDED.ease,
			streak = EXCLUDED.streak,
			interval_days = EXCLUDED.interval_days,
			next_review_at = EXCLUDED.next_review_at,
			review_count = EXCLUDED.review_count,
			correct_count = EXCLUDED.correct_count,
			first_seen_at = COALESCE(user_progress.first_seen_at, EXCLUDED.first_seen_at),
			last_reviewed_at = EXCLUDED.last_reviewed_at
	`

	_, err := r.db.Exec(
		ctx,
		query,
		progress.UserID,
		progress.NameNumber,
		progress.Phase,
		progress.Ease,
		progress.Streak,
		progress.IntervalDays,
		progress.NextReviewAt,
		progress.ReviewCount,
		progress.CorrectCount,
		progress.FirstSeenAt,
		progress.LastReviewedAt,
	)

	if err != nil {
		return fmt.Errorf("upsert progress: %w", err)
	}

	return nil
}

// Get retrieves a single progress record by userID and nameNumber.
func (r *ProgressRepository) Get(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error) {
	query := `
		SELECT user_id, name_number, phase, ease, streak, interval_days,
		       next_review_at, review_count, correct_count, first_seen_at, last_reviewed_at
		FROM user_progress
		WHERE user_id = $1 AND name_number = $2
	`

	var progress entities.UserProgress
	var phase string

	err := r.db.QueryRow(ctx, query, userID, nameNumber).Scan(
		&progress.UserID,
		&progress.NameNumber,
		&phase,
		&progress.Ease,
		&progress.Streak,
		&progress.IntervalDays,
		&progress.NextReviewAt,
		&progress.ReviewCount,
		&progress.CorrectCount,
		&progress.FirstSeenAt,
		&progress.LastReviewedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProgressNotFound
		}
		return nil, fmt.Errorf("get progress: %w", err)
	}

	progress.Phase = entities.Phase(phase)
	return &progress, nil
}

func (r *ProgressRepository) GetByNumbers(ctx context.Context, userID int64, nums []int) (map[int]*entities.UserProgress, error) {
	query := `
      SELECT user_id, name_number, phase, ease, streak, interval_days,
             next_review_at, review_count, correct_count, first_seen_at, last_reviewed_at
      FROM user_progress
      WHERE user_id = $1 AND name_number = ANY($2::int4[])
    `

	numsInt4 := toInt4(nums)
	rows, err := r.db.Query(ctx, query, userID, numsInt4)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[int]*entities.UserProgress, len(nums))
	for rows.Next() {
		p := new(entities.UserProgress)
		if err := rows.Scan(
			&p.UserID, &p.NameNumber, &p.Phase, &p.Ease, &p.Streak, &p.IntervalDays,
			&p.NextReviewAt, &p.ReviewCount, &p.CorrectCount, &p.FirstSeenAt, &p.LastReviewedAt,
		); err != nil {
			return nil, err
		}
		res[p.NameNumber] = p
	}
	return res, rows.Err()
}

func toInt4(nums []int) []int32 {
	out := make([]int32, len(nums))
	for i, v := range nums {
		out[i] = int32(v)
	}
	return out
}

func (r *ProgressRepository) GetStreak(ctx context.Context, userID int64, nameNumber int) (int, error) {
	query := `
		SELECT streak
		FROM user_progress WHERE user_id = $1 AND name_number = $2
	`

	var streak int
	err := r.db.QueryRow(ctx, query, userID, nameNumber).Scan(&streak)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrProgressNotFound
		}
		return 0, fmt.Errorf("get streak: %w", err)
	}

	return streak, nil
}

// GetNamesDueForReview retrieves names that need review based on SRS.
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

	return nameNumbers, rows.Err()
}

// GetLearningNames retrieves names in the learning phase that need practice.
func (r *ProgressRepository) GetLearningNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
			SELECT name_number
			FROM user_progress
			WHERE user_id = $1
			  AND phase = 'learning'
			  AND (next_review_at IS NULL OR next_review_at <= NOW())
			ORDER BY 
				COALESCE(next_review_at, last_reviewed_at) NULLS FIRST
			LIMIT $2
		`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get learning names: %w", err)
	}
	defer rows.Close()

	nameNumbers := make([]int, 0, limit)
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("scan learning name: %w", err)
		}
		nameNumbers = append(nameNumbers, num)
	}

	return nameNumbers, rows.Err()
}

func (r *ProgressRepository) GetNamesForIntroduction(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
		WITH all_names AS (
			SELECT generate_series(1, 99) AS name_number
		)
		SELECT an.name_number
		FROM all_names an
		LEFT JOIN user_progress up ON up.user_id = $1 AND up.name_number = an.name_number
		WHERE up.name_number IS NULL
		ORDER BY an.name_number ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get names for introduction: %w", err)
	}
	defer rows.Close()

	var nameNumbers []int
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("scan introduction name: %w", err)
		}
		nameNumbers = append(nameNumbers, num)
	}

	return nameNumbers, rows.Err()
}

// GetNewNames returns names in "new" phase or early "learning" for quiz introduction.
// Used ONLY in Free mode quizzes to introduce new names.
func (r *ProgressRepository) GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
		WITH all_names AS (
			SELECT generate_series(1, 99) AS name_number
		)
		SELECT an.name_number
		FROM all_names an
		LEFT JOIN user_progress up ON an.name_number = up.name_number AND up.user_id = $1
		WHERE up.name_number IS NULL
		   OR (up.phase = 'new' AND up.review_count < 2)
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

	return nameNumbers, rows.Err()
}

// GetRandomReinforcementNames retrieves random learned names for reinforcement.
func (r *ProgressRepository) GetRandomReinforcementNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
		SELECT name_number
		FROM user_progress
		WHERE user_id = $1
		  AND phase IN ('learning', 'mastered')
		  AND review_count > 0
		ORDER BY RANDOM()
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get random reinforcement names: %w", err)
	}
	defer rows.Close()

	nameNumbers := make([]int, 0, limit)
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("scan reinforcement name: %w", err)
		}
		nameNumbers = append(nameNumbers, num)
	}

	return nameNumbers, rows.Err()
}

// MarkAsIntroduced marks a name as introduced to the user (first time viewing).
func (r *ProgressRepository) MarkAsIntroduced(ctx context.Context, userID int64, nameNumber int) error {
	now := time.Now()
	nextReview := now.Add(24 * time.Hour)

	query := `
		INSERT INTO user_progress (
			user_id, name_number, phase, ease, streak, interval_days,
			next_review_at, review_count, correct_count, 
			first_seen_at, last_reviewed_at, introduced_at, created_at, updated_at
		)
		VALUES ($1, $2, 'new', 2.5, 0, 1, $3, 0, 0, $4, $4, NULL, $4, $4)
		ON CONFLICT (user_id, name_number) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, nameNumber, nextReview, now)
	if err != nil {
		return fmt.Errorf("mark as introduced: %w", err)
	}

	return nil
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
			COUNT(*) FILTER (WHERE phase = 'new') as new_count,
			COUNT(*) FILTER (WHERE phase = 'learning') as learning_count,
			COUNT(*) FILTER (WHERE phase = 'mastered') as mastered_count,
			COUNT(*) FILTER (WHERE next_review_at IS NOT NULL AND next_review_at <= NOW()) as due_today,
			CASE
				WHEN SUM(review_count) > 0 THEN
					LEAST(100, (SUM(correct_count)::float / SUM(review_count)::float) * 100)
				ELSE 0
			END as accuracy,
			MAX(last_reviewed_at) as last_activity,
			COALESCE(AVG(ease), 2.5) as avg_ease
		FROM user_progress
		WHERE user_id = $1
	`

	var stats ProgressStats
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.TotalViewed,
		&stats.NewCount,
		&stats.LearningCount,
		&stats.MasteredCount,
		&stats.DueToday,
		&stats.Accuracy,
		&stats.LastActivityAt,
		&stats.AverageEase,
	)

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	stats.Learned = stats.MasteredCount
	stats.InProgress = stats.NewCount + stats.LearningCount
	stats.NotStarted = 99 - stats.TotalViewed

	if stats.Accuracy > 100 {
		stats.Accuracy = 100
	}

	return &stats, nil
}

// GetNextDueName retrieves the next name due for review.
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
			return 0, nil // No names due
		}
		return 0, fmt.Errorf("get next due name: %w", err)
	}

	return nameNumber, nil
}

// GetByUserID retrieves all progress records for a given user.
func (r *ProgressRepository) GetByUserID(ctx context.Context, userID int64) ([]*entities.UserProgress, error) {
	query := `
    	SELECT user_id, name_number, last_reviewed_at, correct_count,
    	       phase, ease, streak, interval_days, next_review_at, review_count, first_seen_at
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
		var phase string
		err = rows.Scan(
			&p.UserID,
			&p.NameNumber,
			&p.LastReviewedAt,
			&p.CorrectCount,
			&phase,
			&p.Ease,
			&p.Streak,
			&p.IntervalDays,
			&p.NextReviewAt,
			&p.ReviewCount,
			&p.FirstSeenAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		p.Phase = entities.Phase(phase)
		progress = append(progress, &p)
	}

	return progress, rows.Err()
}
