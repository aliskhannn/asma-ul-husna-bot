package service

import (
	"context"
	"errors"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

// ProgressService provides business logic for tracking user progress.
type ProgressService struct {
	repository ProgressRepository
}

// NewProgressService creates a new ProgressService with the provided repository.
func NewProgressService(repository ProgressRepository) *ProgressService {
	return &ProgressService{repository: repository}
}

// MarkAsViewed marks a name as viewed by the user.
func (s *ProgressService) MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error {
	return s.repository.MarkAsViewed(ctx, userID, nameNumber)
}

// RecordReviewSRS updates progress taking SRS (Spaced Repetition System) into account.
// It updates the user's progress based on the answer quality and current time.
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

// ProgressSummary contains a summary of user progress for display.
type ProgressSummary struct {
	Learned        int     // number of learned names
	InProgress     int     // number of names in progress
	NotStarted     int     // number of names not started
	Percentage     float64 // percentage of learned names
	DaysToComplete int     // estimated days to complete learning
	Accuracy       float64 // average accuracy of answers
}

// GetProgressSummary calculates and returns a summary of user progress.
// It uses the user's current stats and daily learning rate to estimate completion.
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
