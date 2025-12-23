package entities

import "time"

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

// ReminderWithUser combines reminder settings with user info and timezone.
type ReminderWithUser struct {
	UserID        int64
	ChatID        int64
	IsEnabled     bool
	IntervalHours int
	StartTime     string
	EndTime       string
	LastSentAt    *time.Time
	NextSendAt    *time.Time
	Timezone      string
}

// UserReminders contains reminder configuration for a user.
type UserReminders struct {
	UserID        int64
	IsEnabled     bool
	IntervalHours int        // interval between reminders (in hours)
	StartTime     string     // format "HH:MM:SS"
	EndTime       string     // format "HH:MM:SS"
	LastSentAt    *time.Time // timestamp of the last sent reminder
	NextSendAt    *time.Time
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
		StartTime:     "08:00:00",
		EndTime:       "20:00:00",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// CalculateNextSendAt calculates the next scheduled reminder time.
// It ensures reminders are sent only at round hours (e.g., 8:00, 9:00).
func (r *UserReminders) CalculateNextSendAt(timezone string, now time.Time) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	userLocalTime := now.In(loc)

	// Parse start hour
	startTime, _ := time.Parse("15:04:05", r.StartTime)
	startHour := startTime.Hour()

	// Parse end hour
	endTime, _ := time.Parse("15:04:05", r.EndTime)
	endHour := endTime.Hour()

	currentHour := userLocalTime.Hour()

	// Find next valid hour
	var nextHour int

	if currentHour < startHour {
		// Before start time today
		nextHour = startHour
	} else if currentHour >= endHour {
		// After end time today, schedule for tomorrow's start
		nextHour = startHour + 24
	} else {
		// Within window, find next interval-aligned hour
		hoursSinceStart := currentHour - startHour
		intervalsNeeded := (hoursSinceStart / r.IntervalHours) + 1
		nextHour = startHour + (intervalsNeeded * r.IntervalHours)

		// If next hour exceeds end time, schedule for tomorrow
		if nextHour > endHour {
			nextHour = startHour + 24
		}
	}

	// Build next send time at the start of the hour (XX:00:00)
	nextSendTime := time.Date(
		userLocalTime.Year(),
		userLocalTime.Month(),
		userLocalTime.Day(),
		nextHour%24,
		0, // minute
		0, // second
		0, // nanosecond
		loc,
	)

	// Add day if needed
	if nextHour >= 24 {
		nextSendTime = nextSendTime.AddDate(0, 0, 1)
	}

	return nextSendTime.In(time.UTC)
}

// CanSendNow checks if it's time to send a reminder.
func (r *ReminderWithUser) CanSendNow(now time.Time) bool {
	if !r.IsEnabled {
		return false
	}

	if r.NextSendAt == nil {
		return true // First reminder, should send
	}

	// Check if current time has reached or passed next_send_at
	return now.After(*r.NextSendAt) || now.Equal(*r.NextSendAt)
}
