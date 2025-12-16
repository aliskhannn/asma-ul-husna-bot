package entities

import "time"

type AnswerQuality string

const (
	QualityFail AnswerQuality = "fail" // incorrect answer
	QualityHard AnswerQuality = "hard" // correct, but difficult
	QualityGood AnswerQuality = "good" // correct, easy
)

// Phase - name learning phase
type Phase string

const (
	PhaseNew      Phase = "new"      // new name
	PhaseLearning Phase = "learning" // in the process of studying
	PhaseMastered Phase = "mastered" // learned
)

type UserProgress struct {
	UserID     int64
	NameNumber int

	// SRS fields.
	Phase        Phase      // the learning phase
	Ease         float64    // ease factor (1.3-2.5)
	Streak       int        // series of correct answers in a row
	IntervalDays int        // current interval in days
	NextReviewAt *time.Time // when to repeat

	// Old fields (maybe will be removed)
	IsLearned      bool
	LastReviewedAt *time.Time // nullable
	CorrectCount   int
}

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

// UpdateSRS - updates SRS parameters after response.
func (p *UserProgress) UpdateSRS(quality AnswerQuality, now time.Time) {
	switch quality {
	case QualityFail:
		p.Streak = 0
		p.Ease = max(1.3, p.Ease-0.2)
		p.IntervalDays = 0
		// Repeat in 10 minutes.
		next := now.Add(10 * time.Minute)
		p.NextReviewAt = &next
		p.Phase = PhaseLearning

	case QualityHard:
		p.Streak++
		p.Ease = max(1.3, p.Ease-0.05)
		p.IntervalDays = calculateIntervalDays(p.Ease, p.Streak)
		// Reduce the interval by 30%
		next := now.Add(time.Duration(float64(p.IntervalDays)*0.7*24) * time.Hour)
		p.NextReviewAt = &next
		if p.Streak >= 3 && p.Phase == PhaseLearning {
			p.Phase = PhaseMastered
		}

	case QualityGood:
		p.Streak++
		p.Ease = min(2.5, p.Ease+0.05)
		p.IntervalDays = calculateIntervalDays(p.Ease, p.Streak)
		next := now.Add(time.Duration(p.IntervalDays*24) * time.Hour)
		p.NextReviewAt = &next
		if p.Streak >= 3 {
			p.Phase = PhaseMastered
		}
	}

	// Updating old fields for compatibility
	p.LastReviewedAt = &now
	if p.Phase == PhaseMastered {
		p.IsLearned = true
	}
}

// calculateIntervalDays calculates the interval in days
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
	// Exponential growth.
	base := float64(streak) * ease
	if base < 1 {
		base = 1
	}
	return int(base * 2)
}
