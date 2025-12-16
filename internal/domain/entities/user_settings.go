package entities

import (
	"fmt"
	"time"
)

type UserSettings struct {
	UserID              int64
	NamesPerDay         int
	QuizLength          int
	QuizMode            string
	ShowTransliteration bool
	ShowAudio           bool
	MaxReviewsPerDay    int
	LanguageCode        *string // nullable, "ru", "en"
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func NewUserSettings(userID int64) *UserSettings {
	now := time.Now()
	return &UserSettings{
		UserID:              userID,
		NamesPerDay:         1,
		QuizLength:          10,
		QuizMode:            "mixed",
		ShowTransliteration: true,
		ShowAudio:           true,
		MaxReviewsPerDay:    50,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

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

func (us *UserSettings) Validate() error {
	if us.NamesPerDay < 1 || us.NamesPerDay > 20 {
		return fmt.Errorf("names_per_day must be between 1 and 20")
	}
	if us.QuizLength < 5 || us.QuizLength > 50 {
		return fmt.Errorf("quiz_length must be between 5 and 50")
	}
	validModes := []string{"new_only", "review_only", "mixed"}
	for _, mode := range validModes {
		if us.QuizMode == mode {
			return nil
		}
	}
	return fmt.Errorf("invalid quiz_mode")
}
