package service

import (
	"context"
	"math/rand"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// QuestionSelector implements smart question selection for quizzes.
type QuestionSelector struct {
	progressRepo  ProgressRepository
	settingsRepo  SettingsRepository
	dailyNameRepo DailyNameRepository
}

// NewQuestionSelector creates a new QuestionSelector.
func NewQuestionSelector(
	progressRepo ProgressRepository,
	settingsRepo SettingsRepository,
	dailyNameRepo DailyNameRepository,
) *QuestionSelector {
	return &QuestionSelector{
		progressRepo:  progressRepo,
		settingsRepo:  settingsRepo,
		dailyNameRepo: dailyNameRepo,
	}
}

// SelectQuestions selects names for a quiz based on SRS priority and quiz mode.
func (s *QuestionSelector) SelectQuestions(
	ctx context.Context,
	userID int64,
	totalQuestions int,
	quizMode string,
) ([]int, error) {
	// Get user settings to check learning mode
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		// Default to guided mode if settings not found
		settings = &entities.UserSettings{LearningMode: "guided"}
	}

	// Selection strategy depends on LEARNING MODE, not quiz mode
	switch settings.LearningMode {
	case "guided":
		return s.selectGuidedMode(ctx, userID, totalQuestions, quizMode)
	case "free":
		return s.selectFreeMode(ctx, userID, totalQuestions, quizMode)
	default:
		return s.selectGuidedMode(ctx, userID, totalQuestions, quizMode)
	}
}

// selectGuidedMode: Shows today's introduced names + due/learning.
func (s *QuestionSelector) selectGuidedMode(
	ctx context.Context,
	userID int64,
	total int,
	quizMode string,
) ([]int, error) {
	var selected []int

	// Priority 1: Due reviews - 40%
	dueLimit := total * 40 / 100
	if dueLimit < 1 {
		dueLimit = 1
	}

	dueNames, err := s.progressRepo.GetNamesDueForReview(ctx, userID, dueLimit)
	if err != nil {
		return nil, err
	}
	selected = append(selected, dueNames...)

	remaining := total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 2: Learning phase - 30%
	learningLimit := total * 30 / 100
	if learningLimit < 1 {
		learningLimit = 1
	}
	if learningLimit > remaining {
		learningLimit = remaining
	}

	learningNames, err := s.progressRepo.GetLearningNames(ctx, userID, learningLimit)
	if err != nil {
		return nil, err
	}
	selected = append(selected, learningNames...)

	remaining = total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 3: Today's introduced names (phase = "new")
	todayNames, err := s.dailyNameRepo.GetTodayNames(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(todayNames) > remaining {
		todayNames = todayNames[:remaining]
	}
	selected = append(selected, todayNames...)

	remaining = total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 4: Random reinforcement
	randomNames, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	selected = append(selected, randomNames...)

	// Shuffle
	rand.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected, nil
}

// selectFreeMode: Can introduce new names in quiz.
func (s *QuestionSelector) selectFreeMode(
	ctx context.Context,
	userID int64,
	total int,
	quizMode string,
) ([]int, error) {
	// Strategy based on quiz_mode
	switch quizMode {
	case "review":
		return s.selectReviewOnly(ctx, userID, total)
	case "new":
		return s.selectNewOnly(ctx, userID, total)
	case "mixed":
		return s.selectMixed(ctx, userID, total)
	default:
		return s.selectMixed(ctx, userID, total)
	}
}

// selectReviewOnly selects only names that are due for review.
func (s *QuestionSelector) selectReviewOnly(ctx context.Context, userID int64, total int) ([]int, error) {
	var selected []int

	// Priority 1: Due reviews (SRS)
	dueNames, err := s.progressRepo.GetNamesDueForReview(ctx, userID, total)
	if err != nil {
		return nil, err
	}
	selected = append(selected, dueNames...)

	remaining := total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 2: Learning phase names
	learningNames, err := s.progressRepo.GetLearningNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	selected = append(selected, learningNames...)

	remaining = total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 3: Random reinforcement
	randomNames, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	selected = append(selected, randomNames...)

	return selected, nil
}

// selectNewOnly selects only new names for introduction.
func (s *QuestionSelector) selectNewOnly(ctx context.Context, userID int64, total int) ([]int, error) {
	var selected []int

	// Priority 1: New names for introduction
	newNames, err := s.progressRepo.GetNamesForIntroduction(ctx, userID, total)
	if err != nil {
		return nil, err
	}
	selected = append(selected, newNames...)

	remaining := total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 2: Learning phase names (if not enough new names)
	learningNames, err := s.progressRepo.GetLearningNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	selected = append(selected, learningNames...)

	return selected, nil
}

// selectMixed selects a balanced mix of due, learning, and new names.
func (s *QuestionSelector) selectMixed(ctx context.Context, userID int64, total int) ([]int, error) {
	var selected []int

	// Priority 1: Due reviews (highest priority) - 40% of total
	dueLimit := total * 40 / 100
	if dueLimit < 1 {
		dueLimit = 1
	}

	dueNames, err := s.progressRepo.GetNamesDueForReview(ctx, userID, dueLimit)
	if err != nil {
		return nil, err
	}
	selected = append(selected, dueNames...)

	remaining := total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 2: Learning phase names - 30% of total
	learningLimit := total * 30 / 100
	if learningLimit < 1 {
		learningLimit = 1
	}
	if learningLimit > remaining {
		learningLimit = remaining
	}

	learningNames, err := s.progressRepo.GetLearningNames(ctx, userID, learningLimit)
	if err != nil {
		return nil, err
	}
	selected = append(selected, learningNames...)

	remaining = total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 3: New names (30%)
	newLimit := remaining
	newNames, err := s.progressRepo.GetNamesForIntroduction(ctx, userID, newLimit)
	if err != nil {
		return nil, err
	}
	selected = append(selected, newNames...)

	remaining = total - len(selected)
	if remaining <= 0 {
		return selected, nil
	}

	// Priority 4: Random reinforcement to fill remaining slots
	randomNames, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	selected = append(selected, randomNames...)

	// Shuffle
	rand.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected, nil
}
