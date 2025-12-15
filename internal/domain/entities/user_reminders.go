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
