package service

import (
	"context"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

// UserRepository defines operations for user persistence.
type UserRepository interface {
	// SaveUser inserts a new user or updates an existing one in the database.
	SaveUser(ctx context.Context, user *entities.User) error
	// UserExists checks if a user with the given ID exists.
	UserExists(ctx context.Context, userID int64) (bool, error)
}

// NameRepository defines operations for accessing Allah's names.
type NameRepository interface {
	// GetByNumber retrieves a name by its number.
	GetByNumber(_ context.Context, number int) (*entities.Name, error)
	// GetRandom retrieves a random name.
	GetRandom(_ context.Context) (*entities.Name, error)
	// GetAll retrieves all names.
	GetAll(_ context.Context) ([]*entities.Name, error)
}

// ProgressRepository defines operations for user progress tracking.
type ProgressRepository interface {
	// MarkAsViewed marks a name as viewed by the user.
	MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error
	// RecordReview records a quiz answer and updates progress.
	RecordReview(ctx context.Context, userID int64, nameNumber int, isCorrect bool, reviewedAt time.Time) error
	// GetNewNames retrieves names that haven't been learned yet.
	GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error)
	// GetNamesDueForReview retrieves names due for review according to SRS.
	GetNamesDueForReview(ctx context.Context, userID int64, limit int) ([]int, error)
	// GetStats returns user progress statistics.
	GetStats(ctx context.Context, userID int64) (*repository.ProgressStats, error)
	// Upsert creates or updates a progress record.
	Upsert(ctx context.Context, progress *entities.UserProgress) error
	// Get retrieves a single progress record.
	Get(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error)
	// GetNextDueName retrieves the next name due for review.
	GetNextDueName(ctx context.Context, userID int64) (int, error)
	// GetOrCreateDailyName retrieves or creates a daily name for the user.
	GetOrCreateDailyName(ctx context.Context, userID int64, dateUTC time.Time, namesPerDay int) (int, error)
	// GetRandomLearnedName retrieves a random learned name.
	GetRandomLearnedName(ctx context.Context, userID int64) (int, error)
	// GetNextDailyName retrieves the next daily name for review.
	GetNextDailyName(ctx context.Context, userID int64, dateUTC time.Time) (int, error)
}

// QuizRepository defines operations for quiz session and answer persistence.
type QuizRepository interface {
	// Create inserts a new quiz session and returns its ID.
	Create(ctx context.Context, s *entities.QuizSession) (int64, error)
	// GetByID retrieves a quiz session by its ID.
	GetByID(ctx context.Context, id int64) (*entities.QuizSession, error)
	// Update updates an existing quiz session.
	Update(ctx context.Context, s *entities.QuizSession) error
	// SaveAnswer saves a quiz answer.
	SaveAnswer(ctx context.Context, a *entities.QuizAnswer) error
}

// SettingsRepository defines operations for user settings persistence.
type SettingsRepository interface {
	// Create creates default settings for a user.
	Create(ctx context.Context, userID int64) error
	// GetByUserID retrieves settings for a user.
	GetByUserID(ctx context.Context, userID int64) (*entities.UserSettings, error)
	// UpdateNamesPerDay updates the number of names to learn per day.
	UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error
	// UpdateQuizMode updates the quiz mode setting.
	UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error
}

// ReminderRepository manages reminder persistence.
type ReminderRepository interface {
	// GetDueReminders retrieves reminders that are due to be sent.
	GetDueReminders(ctx context.Context) ([]*entities.ReminderWithUser, error)
	// MarkAsSent marks a reminder as sent.
	MarkAsSent(ctx context.Context, userID int64, sentAt time.Time) error
	// GetByUserID retrieves reminder settings for a user.
	GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error)
	// Upsert creates or updates reminder settings.
	Upsert(ctx context.Context, rem *entities.UserReminders) error
}

// ReminderNotifier sends reminder notifications to users.
type ReminderNotifier interface {
	// SendReminder sends a reminder message to a user.
	SendReminder(chatID int64, payload entities.ReminderPayload) error
}
