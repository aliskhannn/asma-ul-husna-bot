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

	rng *rand.Rand
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
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectQuestions selects name numbers for a quiz based on SRS priority and the quiz mode.
// Selection strategy depends on the learning mode (guided/free).
func (s *QuestionSelector) SelectQuestions(
	ctx context.Context,
	userID int64,
	total int,
	quizMode string,
) ([]int, error) {
	if total <= 0 {
		return nil, nil
	}

	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil || settings == nil {
		settings = &entities.UserSettings{LearningMode: string(entities.ModeGuided)}
	}

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
		return s.reviewOnly(ctx, userID, total)
	case "mixed":
		return s.guidedMixed(ctx, userID, total)
	default:
		return s.guidedMixed(ctx, userID, total)
	}
}

// guidedNew prioritizes debt (oldest unfinished) and then today's not-mastered names.
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

	today, err := s.dailyNameRepo.GetTodayNames(ctx, userID)
	if err != nil {
		return nil, err
	}

	today, err = s.filterNotMasteredByStreak(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	today = takeFirst(today, remaining)
	out = append(out, today...)

	return uniqueKeepOrder(out), nil
}

// reviewOnly selects due first, then due learning, then reinforcement (mastered and not due).
func (s *QuestionSelector) reviewOnly(ctx context.Context, userID int64, total int) ([]int, error) {
	var out []int

	due, err := s.progressRepo.GetNamesDueForReview(ctx, userID, total)
	if err != nil {
		return nil, err
	}
	out, remaining := appendAndRemaining(out, due, total)
	if remaining == 0 {
		return uniqueKeepOrder(out), nil
	}

	learning, err := s.progressRepo.GetLearningNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out, remaining = appendAndRemaining(out, learning, total)
	if remaining == 0 {
		return uniqueKeepOrder(out), nil
	}

	reinf, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, reinf...)

	return uniqueKeepOrder(out), nil
}

// guidedMixed selects due, then today's not-mastered names, then due learning, then reinforcement.
// The final list is shuffled to mix categories.
func (s *QuestionSelector) guidedMixed(ctx context.Context, userID int64, total int) ([]int, error) {
	var out []int

	dueLimit := calcDueLimit(total)
	due, err := s.progressRepo.GetNamesDueForReview(ctx, userID, dueLimit)
	if err != nil {
		return nil, err
	}
	out, remaining := appendAndRemaining(out, due, total)
	if remaining == 0 {
		return s.shuffled(uniqueKeepOrder(out)), nil
	}

	today, err := s.dailyNameRepo.GetTodayNames(ctx, userID)
	if err != nil {
		return nil, err
	}
	today, err = s.filterNotMasteredByStreak(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	today = takeFirst(today, remaining)
	out, remaining = appendAndRemaining(out, today, total)
	if remaining == 0 {
		return s.shuffled(uniqueKeepOrder(out)), nil
	}

	learningLimit := calcLearningLimit(total, remaining)
	learning, err := s.progressRepo.GetLearningNames(ctx, userID, learningLimit)
	if err != nil {
		return nil, err
	}
	out, remaining = appendAndRemaining(out, learning, total)
	if remaining == 0 {
		return s.shuffled(uniqueKeepOrder(out)), nil
	}

	reinf, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, reinf...)

	return s.shuffled(uniqueKeepOrder(out)), nil
}

// selectFree selects questions for free learning mode based on quiz mode.
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

// freeNew selects new names for introduction (free mode only).
func (s *QuestionSelector) freeNew(ctx context.Context, userID int64, total int) ([]int, error) {
	names, err := s.progressRepo.GetNewNames(ctx, userID, total)
	if err != nil {
		return nil, err
	}
	return uniqueKeepOrder(names), nil
}

// freeMixed selects due, then due learning, then new, then reinforcement and shuffles the result.
func (s *QuestionSelector) freeMixed(ctx context.Context, userID int64, total int) ([]int, error) {
	var out []int

	dueLimit := calcDueLimit(total)
	due, err := s.progressRepo.GetNamesDueForReview(ctx, userID, dueLimit)
	if err != nil {
		return nil, err
	}
	out, remaining := appendAndRemaining(out, due, total)
	if remaining == 0 {
		return s.shuffled(uniqueKeepOrder(out)), nil
	}

	learningLimit := calcLearningLimit(total, remaining)
	learning, err := s.progressRepo.GetLearningNames(ctx, userID, learningLimit)
	if err != nil {
		return nil, err
	}
	out, remaining = appendAndRemaining(out, learning, total)
	if remaining == 0 {
		return s.shuffled(uniqueKeepOrder(out)), nil
	}

	newNames, err := s.progressRepo.GetNewNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out, remaining = appendAndRemaining(out, newNames, total)
	if remaining == 0 {
		return s.shuffled(uniqueKeepOrder(out)), nil
	}

	reinf, err := s.progressRepo.GetRandomReinforcementNames(ctx, userID, remaining)
	if err != nil {
		return nil, err
	}
	out = append(out, reinf...)

	return s.shuffled(uniqueKeepOrder(out)), nil
}

// filterNotMasteredByStreak keeps names that are not mastered according to the streak threshold.
// If progress does not exist, the name is treated as not mastered.
func (s *QuestionSelector) filterNotMasteredByStreak(ctx context.Context, userID int64, nums []int) ([]int, error) {
	out := make([]int, 0, len(nums))
	for _, n := range nums {
		streak, err := s.progressRepo.GetStreak(ctx, userID, n)
		if err != nil {
			out = append(out, n)
			continue
		}
		if streak < entities.MinStreakForMastery {
			out = append(out, n)
		}
	}
	return out, nil
}

// shuffled returns a shuffled copy of the input slice.
func (s *QuestionSelector) shuffled(in []int) []int {
	out := append([]int(nil), in...)
	s.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// uniqueKeepOrder removes duplicates while preserving the original order.
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

// takeFirst returns the first n elements of nums, or the whole slice if it is shorter.
func takeFirst(nums []int, n int) []int {
	if n <= 0 {
		return nil
	}
	if len(nums) <= n {
		return nums
	}
	return nums[:n]
}

// appendAndRemaining appends add to out and returns the updated out and remaining capacity up to total.
func appendAndRemaining(out []int, add []int, total int) ([]int, int) {
	out = append(out, add...)
	rem := total - len(out)
	if rem < 0 {
		rem = 0
	}
	return out, rem
}

// calcDueLimit returns a due quota for mixed mode selection.
func calcDueLimit(total int) int {
	limit := total * 40 / 100
	if limit < 1 {
		limit = 1
	}
	if limit > total {
		limit = total
	}
	return limit
}

// calcLearningLimit returns a learning quota for mixed mode selection.
func calcLearningLimit(total int, remaining int) int {
	limit := total * 30 / 100
	if limit < 1 {
		limit = 1
	}
	if limit > remaining {
		limit = remaining
	}
	return limit
}
