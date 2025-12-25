package entities

import "time"

type ReminderKind string

const (
	ReminderKindNew    ReminderKind = "new"
	ReminderKindReview ReminderKind = "review"
	ReminderKindStudy  ReminderKind = "study"
)

// ReminderPayload is used to build a reminder message payload
// that includes the name to review and related statistics.
type ReminderPayload struct {
	Kind  ReminderKind
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
	LastKind      ReminderKind
	LastSentAt    *time.Time
	NextSendAt    *time.Time
	Timezone      string
}

// UserReminders contains reminder configuration for a user.
type UserReminders struct {
	UserID        int64
	IsEnabled     bool
	IntervalHours int    // interval between reminders (in hours)
	StartTime     string // format "HH:MM:SS"
	EndTime       string // format "HH:MM:SS"
	LastKind      ReminderKind
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
		LastKind:      ReminderKindNew,
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

	userNow := now.In(loc)

	// Парсим как локальное "время дня" пользователя
	startTime, err := time.ParseInLocation("15:04:05", r.StartTime, loc)
	if err != nil {
		// дефолт 08:00:00
		startTime = time.Date(0, 1, 1, 8, 0, 0, 0, loc)
	}
	endTime, err := time.ParseInLocation("15:04:05", r.EndTime, loc)
	if err != nil {
		// дефолт 20:00:00
		endTime = time.Date(0, 1, 1, 20, 0, 0, 0, loc)
	}

	startHour := startTime.Hour()
	endHour := endTime.Hour()

	currentHour := userNow.Hour()

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
	nextSendLocal := time.Date(
		userNow.Year(), userNow.Month(), userNow.Day(),
		nextHour%24, 0, 0, 0,
		loc,
	)

	// Add day if needed
	if nextHour >= 24 {
		nextSendLocal = nextSendLocal.AddDate(0, 0, 1)
	}

	return nextSendLocal.UTC()
}

// CanSendNow checks if it's time to send a reminder.
func (r *ReminderWithUser) CanSendNow(now time.Time) bool {
	if !r.IsEnabled {
		return false
	}

	if r.NextSendAt == nil {
		return true // First reminder, should send
	}

	return !now.Before(*r.NextSendAt)
}
