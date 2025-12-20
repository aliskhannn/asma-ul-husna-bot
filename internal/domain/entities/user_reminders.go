package entities

import "time"

type ReminderWithUser struct {
	UserReminders
	UserID int64 `db:"user_id"`
	ChatID int64 `db:"chat_id"`
}

type ReminderPayload struct {
	Name  Name
	Stats ReminderStats
}

// ReminderStats contains statistics for building reminder message.
type ReminderStats struct {
	DueToday       int
	Learned        int
	NotStarted     int
	DaysToComplete int
}

type UserReminders struct {
	UserID        int64
	IsEnabled     bool
	IntervalHours int        // 1, 2, 3, 4
	StartTimeUTC  string     // "09:00:00" формат HH:MM:SS
	EndTimeUTC    string     // "22:00:00"
	LastSentAt    *time.Time // nullable
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewUserReminders(userID int64) *UserReminders {
	now := time.Now()
	return &UserReminders{
		UserID:        userID,
		IsEnabled:     true,
		IntervalHours: 1,
		StartTimeUTC:  "08:00:00",
		EndTimeUTC:    "20:00:00",
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
