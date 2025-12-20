package telegram

import (
	"context"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

type UserService interface {
	EnsureUser(ctx context.Context, userID, chatID int64) error
}

type NameService interface {
	GetByNumber(ctx context.Context, number int) (*entities.Name, error)
	GetRandom(ctx context.Context) (*entities.Name, error)
	GetAll(ctx context.Context) ([]*entities.Name, error)
}

type ProgressService interface {
	GetProgressSummary(ctx context.Context, userID int64, namesPerDay int) (*service.ProgressSummary, error)
	MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error
	RecordReviewSRS(ctx context.Context, userID int64, nameNumber int, quality entities.AnswerQuality) error
}

type SettingsService interface {
	GetOrCreate(ctx context.Context, userID int64) (*entities.UserSettings, error)
	UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error
	UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error
}

type QuizService interface {
	GenerateQuiz(ctx context.Context, userID int64, mode string) (*entities.QuizSession, []entities.Question, error)
	GetSession(ctx context.Context, sessionID int64) (*entities.QuizSession, error)
	CheckAndSaveAnswer(ctx context.Context, userID int64, session *entities.QuizSession, q *entities.Question, selectedIndex int) (*entities.QuizAnswer, error)
}

type QuizStorage interface {
	Store(sessionID int64, questions []entities.Question)
	Get(sessionID int64) []entities.Question
	Delete(sessionID int64)
}

type ReminderService interface {
	GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error)
	ToggleReminder(ctx context.Context, userID int64) error
	SetReminderIntervalHours(ctx context.Context, userID int64, intervalHours int) error
	SetReminderTimeWindow(ctx context.Context, userID int64, startTime, endTime string) error
	SnoozeReminder(ctx context.Context, userID int64, duration time.Duration) error
	DisableReminder(ctx context.Context, userID int64) error
}
