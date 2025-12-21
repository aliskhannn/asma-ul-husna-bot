package entities

import "time"

// AnswerQuality represents the quality of the user's answer.
type AnswerQuality string

const (
	QualityFail AnswerQuality = "fail" // incorrect answer
	QualityHard AnswerQuality = "hard" // correct, but difficult
	QualityGood AnswerQuality = "good" // correct, easy
)

// Phase represents a learning phase of Allah's names for SRS tracking.
type Phase string

const (
	PhaseNew      Phase = "new"      // new name, not yet studied
	PhaseLearning Phase = "learning" // currently in the learning process
	PhaseMastered Phase = "mastered" // fully memorized and reviewed
)

// UserProgress stores the learning progress of a user for a specific name.
type UserProgress struct {
	UserID     int64
	NameNumber int

	// SRS fields.
	Phase        Phase      // the current learning phase
	Ease         float64    // ease factor (1.3-2.5)
	Streak       int        // number of consecutive correct answers
	IntervalDays int        // current interval to next review, in days
	NextReviewAt *time.Time // timestamp of the next review

	// Legacy fields (for backward compatibility, may be removed later).
	IsLearned      bool
	LastReviewedAt *time.Time // last review timestamp, can be nil
	CorrectCount   int
}

// NewUserProgress creates a new UserProgress instance for a given user and name number.
// It initializes default values for the spaced repetition fields.
func NewUserProgress(userID int64, nameNumber int) *UserProgress {
	return &UserProgress{
		UserID:       userID,
		NameNumber:   nameNumber,
		Phase:        PhaseNew,
		Ease:         2.5,
		Streak:       0,
		IntervalDays: 0,
		IsLearned:    false,
		CorrectCount: 0,
	}
}

// UpdateSRS updates the spaced repetition parameters after the user answers.
//
// It adjusts the user's learning progress depending on the answer quality:
//  1. If the answer is incorrect (QualityFail) — reset streak and ease, reduce interval.
//  2. If the answer is hard (QualityHard) — slightly reduce ease, increase interval, and check mastery.
//  3. If the answer is good (QualityGood) — increase ease and interval, possibly mark as mastered.
//
// The method also updates compatibility fields for legacy data tracking.
func (p *UserProgress) UpdateSRS(quality AnswerQuality, now time.Time) {
	switch quality {
	case QualityFail:
		// 1. Reset progress after a failed attempt
		p.Streak = 0
		p.Ease = max(1.3, p.Ease-0.2)
		p.IntervalDays = 0

		// 2. Schedule a short review in 10 minutes
		next := now.Add(10 * time.Minute)
		p.NextReviewAt = &next
		p.Phase = PhaseLearning

	case QualityHard:
		// 1. Increment streak and slightly reduce ease
		p.Streak++
		p.Ease = max(1.3, p.Ease-0.05)

		// 2. Recalculate interval based on ease and streak
		p.IntervalDays = calculateIntervalDays(p.Ease, p.Streak)

		// 3. Reduce interval by 30% for difficult answers
		next := now.Add(time.Duration(float64(p.IntervalDays)*0.7*24) * time.Hour)
		p.NextReviewAt = &next

		// 4. Promote to mastered phase if progress is sufficient
		if p.Streak >= 3 && p.Phase == PhaseLearning {
			p.Phase = PhaseMastered
		}

	case QualityGood:
		// 1. Increment streak and slightly increase ease
		p.Streak++
		p.Ease = min(2.5, p.Ease+0.05)

		// 2. Calculate interval for the next review
		p.IntervalDays = calculateIntervalDays(p.Ease, p.Streak)

		// 3. Schedule next review after a full interval
		next := now.Add(time.Duration(p.IntervalDays*24) * time.Hour)
		p.NextReviewAt = &next

		// 4. Promote to mastered phase if streak is high
		if p.Streak >= 3 {
			p.Phase = PhaseMastered
		}
	}

	// Update legacy fields for compatibility with older schema
	p.LastReviewedAt = &now
	if p.Phase == PhaseMastered {
		p.IsLearned = true
	}
}

// calculateIntervalDays computes the review interval in days
// based on ease factor and current streak length.
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

	// Exponential interval growth with increasing streak and ease
	base := float64(streak) * ease
	if base < 1 {
		base = 1
	}

	return int(base * 2)
}
