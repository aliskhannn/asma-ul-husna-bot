package service

import (
	"context"
	"errors"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

type ProgressRepository interface {
	MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error
	GetByUserID(ctx context.Context, userID int64) ([]*entities.UserProgress, error)
	RecordReview(ctx context.Context, userID int64, nameNumber int, isCorrect bool, reviewedAt time.Time) error

	GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error)
	HasNewNames(ctx context.Context, userID int64) (bool, error)
	GetNamesToReview(ctx context.Context, userID int64, limit int) ([]int, error)

	GetStats(ctx context.Context, userID int64) (*repository.ProgressStats, error)
	CountLearned(ctx context.Context, userID int64) (int, error)

	Upsert(ctx context.Context, progress *entities.UserProgress) error
	Get(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error)
	GetNamesDueForReview(ctx context.Context, userID int64, limit int) ([]int, error)
	CountDue(ctx context.Context, userID int64) (int, error)

	GetNextDueName(ctx context.Context, userID int64) (int, error)
	GetOrCreateDailyName(ctx context.Context, userID int64, dateUTC time.Time, namesPerDay int) (int, error)
	GetRandomLearnedName(ctx context.Context, userID int64) (int, error)
	GetNextDailyName(ctx context.Context, userID int64, dateUTC time.Time) (int, error)
}

type ProgressService struct {
	repository ProgressRepository
}

func NewProgressService(repository ProgressRepository) *ProgressService {
	return &ProgressService{repository: repository}
}

func (s *ProgressService) MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error {
	return s.repository.MarkAsViewed(ctx, userID, nameNumber)
}

func (s *ProgressService) GetUserProgress(ctx context.Context, userID int64) ([]*entities.UserProgress, error) {
	return s.repository.GetByUserID(ctx, userID)
}

func (s *ProgressService) RecordReview(ctx context.Context, userID int64, nameNumber int, isCorrect bool, reviewedAt time.Time) error {
	return s.repository.RecordReview(ctx, userID, nameNumber, isCorrect, reviewedAt)
}

func (s *ProgressService) GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	return s.repository.GetNewNames(ctx, userID, limit)
}

// RecordReviewSRS updates progress taking SRS into account.
func (s *ProgressService) RecordReviewSRS(ctx context.Context, userID int64, nameNumber int, quality entities.AnswerQuality) error {
	now := time.Now()

	p, err := s.repository.Get(ctx, userID, nameNumber)
	if err != nil && !errors.Is(err, repository.ErrProgressNotFound) {
		return err
	}

	if p == nil {
		p = entities.NewUserProgress(userID, nameNumber)
	}

	p.UpdateSRS(quality, now)

	// Increment the counter only if successful.
	if quality != entities.QualityFail {
		p.CorrectCount++
	}

	return s.repository.Upsert(ctx, p)
}

// GetNamesDueForReview - получить имена на повторение (SRS)
func (s *ProgressService) GetNamesDueForReview(ctx context.Context, userID int64, limit int) ([]int, error) {
	return s.repository.GetNamesDueForReview(ctx, userID, limit)
}

// CountDue - сколько имён нужно повторить
func (s *ProgressService) CountDue(ctx context.Context, userID int64) (int, error) {
	return s.repository.CountDue(ctx, userID)
}

func (s *ProgressService) GetNamesToReview(ctx context.Context, userID int64, limit int) ([]int, error) {
	return s.repository.GetNamesToReview(ctx, userID, limit)
}

func (s *ProgressService) GetStats(ctx context.Context, userID int64) (*repository.ProgressStats, error) {
	return s.repository.GetStats(ctx, userID)
}

func (s *ProgressService) CountLearned(ctx context.Context, userID int64) (int, error) {
	return s.repository.CountLearned(ctx, userID)
}

type ProgressSummary struct {
	Learned        int
	InProgress     int
	NotStarted     int
	Percentage     float64
	DaysToComplete int
	Accuracy       float64
}

func (s *ProgressService) GetProgressSummary(ctx context.Context, userID int64, namesPerDay int) (*ProgressSummary, error) {
	stats, err := s.repository.GetStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	remaining := 99 - stats.Learned
	daysToComplete := 0
	if namesPerDay > 0 && remaining > 0 {
		daysToComplete = (remaining + namesPerDay - 1) / namesPerDay
	}

	percentage := float64(stats.Learned) / 99.0 * 100

	return &ProgressSummary{
		Learned:        stats.Learned,
		InProgress:     stats.InProgress,
		NotStarted:     stats.NotStarted,
		Percentage:     percentage,
		DaysToComplete: daysToComplete,
		Accuracy:       stats.Accuracy,
	}, nil
}
