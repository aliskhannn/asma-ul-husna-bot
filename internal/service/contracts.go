package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres/repository"
)

// UserRepository defines operations for user persistence.
type UserRepository interface {
	// Save inserts a new user or updates an existing one in the database.
	Save(ctx context.Context, user *entities.User) (bool, error)
	// Exists checks if a user with the given ID exists.
	Exists(ctx context.Context, userID int64) (bool, error)
}

// NameRepository defines operations for accessing Allah's names.
type NameRepository interface {
	// GetByNumber retrieves a name by its number.
	GetByNumber(number int) (*entities.Name, error)
	// GetRandom retrieves a random name.
	GetRandom() (*entities.Name, error)
	// GetAll retrieves all names.
	GetAll() ([]*entities.Name, error)
	GetByNumbers(numbers []int) ([]entities.Name, error)
}

// ProgressRepository defines operations for user progress tracking.
type ProgressRepository interface {
	// GetNamesDueForReview retrieves names due for review according to SRS.
	GetNamesDueForReview(ctx context.Context, userID int64, limit int) ([]int, error)
	// GetStats returns user progress statistics.
	GetStats(ctx context.Context, userID int64) (*repository.ProgressStats, error)
	// Get retrieves a single progress record.
	Get(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error)
	// GetNextDueName retrieves the next name due for review.
	GetNextDueName(ctx context.Context, userID int64) (int, error)
	GetNamesForIntroduction(ctx context.Context, userID int64, limit int) ([]int, error)
	MarkAsIntroduced(ctx context.Context, userID int64, nameNumber int) error
	GetLearningNames(ctx context.Context, userID int64, limit int) ([]int, error)
	GetRandomReinforcementNames(ctx context.Context, userID int64, limit int) ([]int, error)
	Upsert(ctx context.Context, progress *entities.UserProgress) error
	GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error)
	GetStreak(ctx context.Context, userID int64, nameNumber int) (int, error)
	GetByNumbers(ctx context.Context, userID int64, nums []int) (map[int]*entities.UserProgress, error)
	CountIntroducedOnDate(ctx context.Context, userID int64, tz string) (int, error)
}

// QuizRepository defines operations for quiz session and answer persistence.
type QuizRepository interface {
	AbandonOldSessions(ctx context.Context, userID int64) error
	Create(ctx context.Context, session *entities.QuizSession) (int64, error)
	CreateQuestion(ctx context.Context, session *entities.QuizQuestion) (int64, error)
	GetSessionForUpdate(ctx context.Context, sessionID, userID int64) (*entities.QuizSession, error)
	GetQuestionByOrder(ctx context.Context, sessionID int64, order int) (*entities.QuizQuestion, error)
	SaveAnswer(ctx context.Context, answer *entities.QuizAnswer) error
	UpdateSession(ctx context.Context, session *entities.QuizSession) error
	GetActiveSessionByUserID(ctx context.Context, userID int64) (*entities.QuizSession, error)
	IsFirstQuiz(ctx context.Context, userID int64) (bool, error)
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
	UpdateLearningMode(ctx context.Context, userID int64, learningMode string) error
	UpsertDefaults(ctx context.Context, userID int64) error
}

// ReminderRepository manages reminder persistence.
type ReminderRepository interface {
	// MarkAsSent marks a reminder as sent.
	MarkAsSent(ctx context.Context, userID int64, sentAt time.Time) error
	// GetByUserID retrieves reminder settings for a user.
	GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error)
	// Upsert creates or updates reminder settings.
	Upsert(ctx context.Context, rem *entities.UserReminders) error
	GetDueReminder(ctx context.Context, userID int64) (*entities.ReminderWithUser, error)
	GetDueRemindersBatch(ctx context.Context, now time.Time, limit, offset int) ([]*entities.ReminderWithUser, error)
	UpdateAfterSend(ctx context.Context, userID int64, sentAt time.Time, nextSendAt time.Time, lastKind entities.ReminderKind) error
	RescheduleNext(ctx context.Context, userID int64, nextSendAt time.Time) error
}

// ReminderNotifier sends reminder notifications to users.
type ReminderNotifier interface {
	// SendReminder sends a reminder message to a user.
	SendReminder(chatID int64, payload entities.ReminderPayload) error
}

type DailyNameRepository interface {
	GetTodayNames(ctx context.Context, userID int64) ([]int, error)
	GetTodayNamesCount(ctx context.Context, userID int64) (int, error)
	HasUnfinishedDays(ctx context.Context, userID int64) (bool, error)
	AddTodayName(ctx context.Context, userID int64, nameNumber int) error
	RemoveTodayName(ctx context.Context, userID int64, nameNumber int) error
	GetOldestUnfinishedName(ctx context.Context, userID int64) (int, error)
}

type ResetRepository interface {
	ResetUser(ctx context.Context, userID int64) error
}

type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
}
