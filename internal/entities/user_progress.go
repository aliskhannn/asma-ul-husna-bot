package entities

import "time"

type UserProgress struct {
	UserID         int64
	NameNumber     int
	IsLearned      bool
	LastReviewedAt *time.Time // nullable
	CorrectCount   int
}

func NewUserProgress(userID int64, nameNumber int) *UserProgress {
	return &UserProgress{
		UserID:       userID,
		NameNumber:   nameNumber,
		IsLearned:    false,
		CorrectCount: 0,
	}
}
