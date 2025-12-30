package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres/repository"
)

// ProgressService provides business logic for tracking user progress.
type ProgressService struct {
	progressRepo ProgressRepository
	settingsRepo SettingsRepository
}

// NewProgressService creates a new ProgressService.
func NewProgressService(
	progressRepo ProgressRepository,
	settingsRepo SettingsRepository,
) *ProgressService {
	return &ProgressService{
		progressRepo: progressRepo,
		settingsRepo: settingsRepo,
	}
}

// ProgressSummary contains a summary of user progress for display.
type ProgressSummary struct {
	Learned        int
	InProgress     int
	NotStarted     int
	Percentage     float64
	DaysToComplete int
	Accuracy       float64
	DueToday       int
	NewCount       int
	LearningCount  int
	MasteredCount  int
}

// GetProgressSummary calculates and returns a summary of user progress.
func (s *ProgressService) GetProgressSummary(ctx context.Context, userID int64) (*ProgressSummary, error) {
	stats, err := s.progressRepo.GetStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, repository.ErrSettingsNotFound) {
			return nil, fmt.Errorf("get settings: %w", err)
		}
		settings = entities.NewUserSettings(userID)
	}

	learned := stats.Learned
	inProgress := stats.InProgress
	notStarted := stats.NotStarted

	percentage := float64(learned) / 99.0 * 100
	daysToComplete := settings.DaysToComplete(learned)

	return &ProgressSummary{
		Learned:        learned,
		InProgress:     inProgress,
		NotStarted:     notStarted,
		Percentage:     percentage,
		DaysToComplete: daysToComplete,
		Accuracy:       stats.Accuracy,
		DueToday:       stats.DueToday,
		NewCount:       stats.NewCount,
		LearningCount:  stats.LearningCount,
		MasteredCount:  stats.MasteredCount,
	}, nil
}

// GetProgress retrieves progress for a specific name.
func (s *ProgressService) GetProgress(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error) {
	progress, err := s.progressRepo.Get(ctx, userID, nameNumber)
	if err != nil {
		if errors.Is(err, repository.ErrProgressNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get progress: %w", err)
	}

	return progress, nil
}

func (s *ProgressService) GetByNumbers(ctx context.Context, userID int64, nums []int) (map[int]*entities.UserProgress, error) {
	return s.progressRepo.GetByNumbers(ctx, userID, nums)
}

func (s *ProgressService) GetStreak(ctx context.Context, userID int64, nameNumber int) (int, error) {
	streak, err := s.progressRepo.GetStreak(ctx, userID, nameNumber)
	if err != nil {
		if errors.Is(err, repository.ErrProgressNotFound) {
			return 0, nil
		}
	}
	return streak, nil
}

// GetDueNames retrieves all names that are due for review.
func (s *ProgressService) GetDueNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	names, err := s.progressRepo.GetNamesDueForReview(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get due names: %w", err)
	}

	return names, nil
}

// GetLearningNames retrieves names in the learning phase.
func (s *ProgressService) GetLearningNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	names, err := s.progressRepo.GetLearningNames(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get learning names: %w", err)
	}

	return names, nil
}

// GetNewNames retrieves new names for introduction.
func (s *ProgressService) GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error) {
	names, err := s.progressRepo.GetNamesForIntroduction(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get new names: %w", err)
	}

	return names, nil
}
