package entities

import "time"

// ReminderWithUser represents reminder data joined with user information.
type ReminderWithUser struct {
	UserReminders
	UserID int64 `db:"user_id"`
	ChatID int64 `db:"chat_id"`
}

// ReminderPayload is used to build a reminder message payload
// that includes the name to review and related statistics.
type ReminderPayload struct {
	Name  Name
	Stats ReminderStats
}

// ReminderStats contains user progress statistics
// that help in forming the reminder message.
type ReminderStats struct {
	DueToday       int // number of names due today
	Learned        int // number of mastered names
	NotStarted     int // number of unstarted names
	DaysToComplete int // estimated days left to complete learning
}

// UserReminders contains reminder configuration for a user.
type UserReminders struct {
	UserID        int64
	IsEnabled     bool
	IntervalHours int        // interval between reminders (in hours)
	StartTimeUTC  string     // start time in UTC, format "HH:MM:SS"
	EndTimeUTC    string     // end time in UTC, format "HH:MM:SS"
	LastSentAt    *time.Time // timestamp of the last sent reminder
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewUserReminders creates a new default reminder configuration for a user.
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

// ParseStartTime parses the StartTimeUTC field into a time.Time object.
func (ur *UserReminders) ParseStartTime() (time.Time, error) {
	return time.Parse("15:04:05", ur.StartTimeUTC)
}

// ParseEndTime parses the EndTimeUTC field into a time.Time object.
func (ur *UserReminders) ParseEndTime() (time.Time, error) {
	return time.Parse("15:04:05", ur.EndTimeUTC)
}

// CanSendNow checks if a reminder can be sent at the current time.
// It validates three conditions:
//  1. Reminders must be enabled.
//  2. The current time must fall between the start and end UTC time.
//  3. Enough time must have passed since the last sent reminder.
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
