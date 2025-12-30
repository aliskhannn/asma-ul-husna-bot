package entities

import "time"

// AnswerQuality represents the quality of the user's answer.
type AnswerQuality string

const (
	QualityFail AnswerQuality = "fail" // incorrect answer
	QualityGood AnswerQuality = "good" // correct, easy
)

// Phase represents a learning phase of Allah's names for SRS tracking.
type Phase string

const (
	PhaseNew      Phase = "new"      // new name, not yet studied
	PhaseLearning Phase = "learning" // currently in the learning process
	PhaseMastered Phase = "mastered" // fully memorized and reviewed
)

// SRS thresholds
const (
	MinStreakForLearning  = 3   // Streak to move from 'new' to 'learning'
	MinStreakForMastery   = 7   // Streak to move to 'mastered'
	MinIntervalForMastery = 21  // Days interval required for mastery
	MaxIntervalDays       = 180 // Cap at 6 months
)

// UserProgress stores the learning progress of a user for a specific name.
type UserProgress struct {
	UserID     int64
	NameNumber int

	// SRS fields
	Phase        Phase
	Ease         float64
	Streak       int
	IntervalDays int
	NextReviewAt *time.Time

	// Tracking fields
	ReviewCount    int
	CorrectCount   int
	FirstSeenAt    *time.Time
	IntroducedAt   *time.Time
	LastReviewedAt *time.Time
}

// NewUserProgress creates a new UserProgress instance for a given user and name number.
// It initializes default values for the spaced repetition fields.
func NewUserProgress(userID int64, nameNumber int) *UserProgress {
	now := time.Now()
	return &UserProgress{
		UserID:       userID,
		NameNumber:   nameNumber,
		Phase:        PhaseNew,
		Ease:         2.5,
		Streak:       0,
		IntervalDays: 0,
		ReviewCount:  0,
		CorrectCount: 0,
		FirstSeenAt:  &now,
	}
}

// UpdateSRS updates the spaced repetition parameters after the user answers.
// It adjusts the user's learning progress based on answer quality using SM-2 algorithm.
func (p *UserProgress) UpdateSRS(quality AnswerQuality, now time.Time) {
	p.ReviewCount++
	p.LastReviewedAt = &now

	switch quality {
	case QualityFail:
		// Reset streak and reduce ease
		p.Streak = 0
		p.Ease = max(1.3, p.Ease-0.2)
		p.IntervalDays = 0

		// Schedule immediate review (10 minutes)
		next := now.Add(10 * time.Minute)
		p.NextReviewAt = &next

		// Demote from mastered if applicable
		if p.Phase == PhaseMastered {
			p.Phase = PhaseLearning
		}

	case QualityGood:
		p.Streak++
		p.CorrectCount++
		p.Ease = min(2.5, p.Ease+0.01)

		p.IntervalDays = calculateIntervalDays(p.Ease, p.Streak)

		next := now.Add(time.Duration(p.IntervalDays) * 24 * time.Hour)
		p.NextReviewAt = &next

		p.updatePhase()
	}
}

// updatePhase transitions between learning phases based on streak and interval.
func (p *UserProgress) updatePhase() {
	if p.Streak >= MinStreakForMastery && p.IntervalDays >= MinIntervalForMastery {
		p.Phase = PhaseMastered
		return
	}

	if p.Phase == PhaseNew && (p.Streak >= MinStreakForLearning || p.ReviewCount >= 2) {
		p.Phase = PhaseLearning
		return
	}
}

// IsLearned returns true if the name is considered learned (mastered).
func (p *UserProgress) IsLearned() bool {
	return p.Phase == PhaseMastered
}

// Accuracy returns the percentage of correct answers.
func (p *UserProgress) Accuracy() float64 {
	if p.ReviewCount == 0 {
		return 0
	}
	return float64(p.CorrectCount) / float64(p.ReviewCount) * 100
}

// calculateIntervalDays computes the review interval in days based on ease factor
// and current streak length using the SM-2 algorithm.
func calculateIntervalDays(ease float64, streak int) int {
	if streak <= 0 {
		return 0
	}
	if streak == 1 {
		return 1
	}
	if streak == 2 {
		return 3
	}
	if streak == 3 {
		return 7
	}

	// SM-2 formula for streak > 3.
	base := 7.0
	for i := 4; i <= streak; i++ {
		base *= ease
	}

	interval := int(base)
	if interval > MaxIntervalDays {
		return MaxIntervalDays
	}
	return interval
}

// DetermineQuality determines answer quality based on correctness and attempt.
func DetermineQuality(isCorrect bool, isFirstAttempt bool) AnswerQuality {
	if !isCorrect {
		return QualityFail
	}
	return QualityGood
}
