package entities

import "time"

type UserReminders struct {
	UserID        int64
	IsEnabled     bool
	Frequency     string // "daily", "twice_daily", "custom"
	StartTimeUTC  string // "09:00:00" формат HH:MM:SS
	EndTimeUTC    string // "22:00:00"
	IntervalHours int
	LastSentAt    *time.Time // nullable
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewUserReminders(userID int64) *UserReminders {
	now := time.Now()
	return &UserReminders{
		UserID:        userID,
		IsEnabled:     true,
		Frequency:     "daily",
		StartTimeUTC:  "08:00:00",
		EndTimeUTC:    "20:00:00",
		IntervalHours: 24,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (ur *UserReminders) ParseStartTime() (time.Time, error) {
	return time.Parse("15:04:05", ur.StartTimeUTC)
}

func (ur *UserReminders) ParseEndTime() (time.Time, error) {
	return time.Parse("15:04:05", ur.EndTimeUTC)
}

func (ur *UserReminders) CanSendNow() bool {
	if !ur.IsEnabled {
		return false
	}

	now := time.Now().UTC()
	currentTime := now.Format("15:04:05")

	if currentTime < ur.StartTimeUTC || currentTime > ur.EndTimeUTC {
		return false
	}

	if ur.LastSentAt == nil {
		return true
	}

	hoursSince := time.Since(*ur.LastSentAt).Hours()
	return hoursSince >= float64(ur.IntervalHours)
}
