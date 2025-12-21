package entities

import (
	"time"
)

// UserSettings stores user-specific configuration and preferences for learning.
type UserSettings struct {
	UserID           int64
	NamesPerDay      int     // number of new names to learn per day
	QuizMode         string  // quiz type: "new", "review", "mixed"
	MaxReviewsPerDay int     // maximum number of reviews allowed per day
	LanguageCode     *string // nullable, defines language code ("ru", "en")
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewUserSettings creates a new UserSettings instance with default values.
func NewUserSettings(userID int64) *UserSettings {
	now := time.Now()
	return &UserSettings{
		UserID:           userID,
		NamesPerDay:      1,
		QuizMode:         "mixed",
		MaxReviewsPerDay: 50,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// DaysToComplete calculates the estimated number of days
// required to learn all 99 names based on current progress.
func (us *UserSettings) DaysToComplete(learnedCount int) int {
	remaining := 99 - learnedCount
	if remaining <= 0 {
		return 0
	}

	days := remaining / us.NamesPerDay
	if remaining%us.NamesPerDay != 0 {
		days++
	}
	return days
}
