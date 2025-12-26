package service

import (
	"context"
	"math/rand"
	"time"

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
	total int,
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
	case string(entities.ModeFree):
		return s.selectFree(ctx, userID, total, quizMode)
	case string(entities.ModeGuided):
		return s.selectGuided(ctx, userID, total, quizMode)
	default:
		return s.selectGuided(ctx, userID, total, quizMode)
	}
}

func (s *QuestionSelector) selectGuided(ctx context.Context, userID int64, total int, quizMode string) ([]int, error) {
	switch quizMode {
	case "new":
		return s.guidedNew(ctx, userID, total)
	case "review":
		return s.reviewOnly(ctx, userID, total) // одинаково для guided/free
	case "mixed":
		return s.guidedMixed(ctx, userID, total)
	default:
		return s.guidedMixed(ctx, userID, total)
	}
}

func (s *QuestionSelector) guidedNew(ctx context.Context, userID int64, total int) ([]int, error) {
	var out []int

	hasDebt, err := s.dailyNameRepo.HasUnfinishedDays(ctx, userID)
	if err != nil {
		return nil, err
	}
	if hasDebt && len(out) < total {
		n, err := s.dailyNameRepo.GetOldestUnfinishedName(ctx, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}

	remaining := total - len(out)
	if remaining <= 0 {
		return uniqueKeepOrder(out), nil
	}

	// 2) today: берём только НЕ выученные
	today, err := s.dailyNameRepo.GetTodayNames(ctx, userID)
	if err != nil {
		return nil, err
	}

	filtered := make([]int, 0, len(today))
	for _, n := range today {
		streak, err := s.progressRepo.GetStreak(ctx, userID, n)
		if err != nil {
			// нет progress => считаем не выучено
			filtered = append(filtered, n)
			continue
		}
		if streak < 7 {
			filtered = append(filtered, n)
		}
	}

	if len(filtered) > remaining {
		filtered = filtered[:remaining]
	}
	out = append(out, filtered...)

	return uniqueKeepOrder(out), nil
}

// REVIEW-only (для guided/free): due -> learning -> reinforcement.
func (s *QuestionSelector) reviewOnly(ctx context.Context, userID int64, total int) ([]int, error) {
	var out []int

	due, err := s.progressRepo.GetNamesDueForReview(ctx, userID, total)
	if err != nil {
		return nil, err
	}
	out = append(out, due...)

	remaining := total - len(out)
	if remaining <= 0 {
		return uniqueKeepOrder(out), nil
	}

	learning, err := s.progressRepo.GetLearningNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, learning...)

	remaining = total - len(out)
	if remaining <= 0 {
		return uniqueKeepOrder(out), nil
	}

	reinf, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, reinf...)

	return uniqueKeepOrder(out), nil
}

// Guided mixed: due + learning + today + reinforcement, затем shuffle.
func (s *QuestionSelector) guidedMixed(ctx context.Context, userID int64, total int) ([]int, error) {
	var out []int

	dueLimit := max(1, total*40/100)
	due, err := s.progressRepo.GetNamesDueForReview(ctx, userID, dueLimit)
	if err != nil {
		return nil, err
	}
	out = append(out, due...)

	remaining := total - len(out)
	if remaining <= 0 {
		return shuffled(uniqueKeepOrder(out)), nil
	}

	learningLimit := min(max(1, total*30/100), remaining)
	learning, err := s.progressRepo.GetLearningNames(ctx, userID, learningLimit)
	if err != nil {
		return nil, err
	}
	out = append(out, learning...)

	remaining = total - len(out)
	if remaining <= 0 {
		return shuffled(uniqueKeepOrder(out)), nil
	}

	today, err := s.dailyNameRepo.GetTodayNames(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(today) > remaining {
		today = today[:remaining]
	}
	out = append(out, today...)

	remaining = total - len(out)
	if remaining <= 0 {
		return shuffled(uniqueKeepOrder(out)), nil
	}

	reinf, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, reinf...)

	return shuffled(uniqueKeepOrder(out)), nil
}

// Free selection: учитывает quizMode; NEW действительно вводит новые через GetNewNames.
func (s *QuestionSelector) selectFree(ctx context.Context, userID int64, total int, quizMode string) ([]int, error) {
	switch quizMode {
	case "review":
		return s.reviewOnly(ctx, userID, total)
	case "new":
		return s.freeNew(ctx, userID, total)
	case "mixed":
		return s.freeMixed(ctx, userID, total)
	default:
		return s.freeMixed(ctx, userID, total)
	}
}

func (s *QuestionSelector) freeNew(ctx context.Context, userID int64, total int) ([]int, error) {
	names, err := s.progressRepo.GetNewNames(ctx, userID, total)
	if err != nil {
		return nil, err
	}
	return uniqueKeepOrder(names), nil
}

func (s *QuestionSelector) freeMixed(ctx context.Context, userID int64, total int) ([]int, error) {
	var out []int

	dueLimit := max(1, total*40/100)
	due, err := s.progressRepo.GetNamesDueForReview(ctx, userID, dueLimit)
	if err != nil {
		return nil, err
	}
	out = append(out, due...)

	remaining := total - len(out)
	if remaining <= 0 {
		return shuffled(uniqueKeepOrder(out)), nil
	}

	learningLimit := min(max(1, total*30/100), remaining)
	learning, err := s.progressRepo.GetLearningNames(ctx, userID, learningLimit)
	if err != nil {
		return nil, err
	}
	out = append(out, learning...)

	remaining = total - len(out)
	if remaining <= 0 {
		return shuffled(uniqueKeepOrder(out)), nil
	}

	newNames, err := s.progressRepo.GetNewNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, newNames...)

	remaining = total - len(out)
	if remaining <= 0 {
		return shuffled(uniqueKeepOrder(out)), nil
	}

	reinf, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, reinf...)

	return shuffled(uniqueKeepOrder(out)), nil
}

// helpers

func shuffled(in []int) []int {
	out := append([]int(nil), in...)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func uniqueKeepOrder(nums []int) []int {
	seen := make(map[int]struct{}, len(nums))
	out := make([]int, 0, len(nums))
	for _, n := range nums {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
