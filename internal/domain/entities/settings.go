package entities

import (
	"time"
)

// LearningMode represents the learning mode setting
type LearningMode string

const (
	ModeGuided LearningMode = "guided"
	ModeFree   LearningMode = "free"
)

// UserSettings stores user-specific configuration and preferences for learning.
type UserSettings struct {
	UserID           int64
	NamesPerDay      int    // number of new names to learn per day
	MaxReviewsPerDay int    // maximum number of reviews allowed per day
	QuizMode         string // quiz type: "new", "review", "mixed"
	LearningMode     string
	LanguageCode     string // "ru", "en"
	Timezone         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewUserSettings creates a new UserSettings instance with default values.
func NewUserSettings(userID int64) *UserSettings {
	now := time.Now()
	return &UserSettings{
		UserID:           userID,
		NamesPerDay:      1,
		MaxReviewsPerDay: 50,
		QuizMode:         "mixed",
		LearningMode:     "guided",
		LanguageCode:     "ru",
		Timezone:         "UTC",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// DaysToComplete estimates days to complete learning based on current progress.
func (s *UserSettings) DaysToComplete(learnedCount int) int {
	if s.NamesPerDay < 0 {
		return 0
	}
	remaining := 99 - learnedCount
	if remaining <= 0 {
		return 0
	}
	return (remaining-1)/s.NamesPerDay + 1
}
