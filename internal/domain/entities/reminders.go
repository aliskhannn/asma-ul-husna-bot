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
		IsEnabled:     false,
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
func (r *UserReminders) CalculateNextSendAt(timezone string, nowUTC time.Time) time.Time {
	loc, err := ParseTimezoneLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	userNow := nowUTC.In(loc)

	y, m, d := userNow.Date()

	startTOD, _ := time.Parse("15:04:05", r.StartTime)
	endTOD, _ := time.Parse("15:04:05", r.EndTime)

	startLocal := time.Date(y, m, d, startTOD.Hour(), startTOD.Minute(), startTOD.Second(), 0, loc)
	endLocal := time.Date(y, m, d, endTOD.Hour(), endTOD.Minute(), endTOD.Second(), 0, loc)

	if !endLocal.After(startLocal) {
		startLocal = time.Date(y, m, d, 8, 0, 0, 0, loc)
		endLocal = time.Date(y, m, d, 20, 0, 0, 0, loc)
	}

	interval := time.Duration(r.IntervalHours) * time.Hour
	if interval <= 0 {
		interval = time.Hour
	}

	if userNow.Before(startLocal) {
		return startLocal.UTC()
	}

	if !userNow.Before(endLocal) {
		return startLocal.AddDate(0, 0, 1).UTC()
	}

	elapsed := userNow.Sub(startLocal)
	k := elapsed / interval
	next := startLocal.Add((k + 1) * interval)

	if !next.Before(endLocal) {
		next = startLocal.AddDate(0, 0, 1)
	}
	
	next = next.Truncate(time.Second)
	return next.UTC()
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
