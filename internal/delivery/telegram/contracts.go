package telegram

import (
	"context"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/storage"
)

// UserService interface for user-related operations.
type UserService interface {
	EnsureUser(ctx context.Context, userID, chatID int64) (bool, error)
	Exists(ctx context.Context, userID int64) (bool, error)
}

// NameService interface for name-related operations.
type NameService interface {
	GetByNumber(ctx context.Context, number int) (*entities.Name, error)
	GetRandom(ctx context.Context) (*entities.Name, error)
	GetAll(ctx context.Context) ([]*entities.Name, error)
}

// ProgressService interface for progress-related operations.
type ProgressService interface {
	GetProgressSummary(ctx context.Context, userID int64) (*service.ProgressSummary, error)
	//IntroduceName(ctx context.Context, userID int64, nameNumber int) error
	GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error)
	GetStreak(ctx context.Context, userID int64, nameNumber int) (int, error)
	GetByNumbers(ctx context.Context, userID int64, nums []int) (map[int]*entities.UserProgress, error)
	//CountIntroducedToday(ctx context.Context, userID int64, tz string) (int, error)
}

// SettingsService interface for settings-related operations.
type SettingsService interface {
	GetOrCreate(ctx context.Context, userID int64) (*entities.UserSettings, error)
	UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error
	UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error
	UpdateLearningMode(ctx context.Context, userID int64, learningMode string) error
	UpdateTimezone(ctx context.Context, userID int64, timezone string) error
}

// QuizService interface for quiz-related operations.
type QuizService interface {
	GetActiveSession(ctx context.Context, userID int64) (*entities.QuizSession, error)
	GetCurrentQuestion(ctx context.Context, sessionID int64, questionNum int) (*entities.QuizQuestion, *entities.Name, error)
	StartQuizSession(ctx context.Context, userID int64, totalQuestions int) (*entities.QuizSession, []entities.Name, error)
	SubmitAnswer(ctx context.Context, sessionID int64, userID int64, selectedOption string) (*service.AnswerResult, error)
	IsFirstQuiz(ctx context.Context, userID int64) (bool, error)
}

// ReminderService interface for reminder-related operations.
type ReminderService interface {
	GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error)
	GetOrCreate(ctx context.Context, userID int64) (*entities.UserReminders, error)
	ToggleReminder(ctx context.Context, userID int64) error
	SetReminderIntervalHours(ctx context.Context, userID int64, intervalHours int) error
	SetReminderTimeWindow(ctx context.Context, userID int64, startTime, endTime string) error
	SnoozeReminder(ctx context.Context, userID int64, duration time.Duration) error
	DisableReminder(ctx context.Context, userID int64) error
}

type DailyNameService interface {
	GetTodayNames(ctx context.Context, userID int64) ([]int, error)
	GetTodayNamesCount(ctx context.Context, userID int64) (int, error)
	AddTodayName(ctx context.Context, userID int64, nameNumber int) error
	GetOldestUnfinishedName(ctx context.Context, userID int64) (int, error)
	HasUnfinishedDays(ctx context.Context, userID int64) (bool, error)
	EnsureTodayPlan(ctx context.Context, userID int64, tz string, namesPerDay int) error
	GetTodayNamesTZ(ctx context.Context, userID int64, tz string) ([]int, error)
	AddTodayNameTZ(ctx context.Context, userID int64, tz string, nameNumber int) error
}

// QuizStorage interface for quiz session storage.
type QuizStorage interface {
	Store(sessionID int64, names []entities.Name)
	Get(sessionID int64) []entities.Name
	Delete(sessionID int64)
	StoreMessageID(sessionID int64, messageID int)
	GetMessageID(sessionID int64) (int, bool)
	DeleteMessageID(sessionID int64)
}

type ReminderStorage interface {
	Store(userID int64, chatID int64, messageID int)
	Get(userID int64) (storage.ReminderMessage, bool)
	Delete(userID int64)
}

type ResetService interface {
	ResetUser(ctx context.Context, userID int64) error
}
