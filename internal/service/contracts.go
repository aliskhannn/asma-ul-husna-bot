package service

import (
	"context"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

type UserRepository interface {
	SaveUser(ctx context.Context, user *entities.User) error
	UserExists(ctx context.Context, userID int64) (bool, error)
}

type NameRepository interface {
	GetByNumber(_ context.Context, number int) (*entities.Name, error)
	GetRandom(_ context.Context) (*entities.Name, error)
	GetAll(_ context.Context) ([]*entities.Name, error)
}

type ProgressRepository interface {
	MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error
	RecordReview(ctx context.Context, userID int64, nameNumber int, isCorrect bool, reviewedAt time.Time) error
	GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error)
	GetNamesDueForReview(ctx context.Context, userID int64, limit int) ([]int, error)
	GetStats(ctx context.Context, userID int64) (*repository.ProgressStats, error)
	Upsert(ctx context.Context, progress *entities.UserProgress) error
	Get(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error)
	GetNextDueName(ctx context.Context, userID int64) (int, error)
	GetOrCreateDailyName(ctx context.Context, userID int64, dateUTC time.Time, namesPerDay int) (int, error)
	GetRandomLearnedName(ctx context.Context, userID int64) (int, error)
	GetNextDailyName(ctx context.Context, userID int64, dateUTC time.Time) (int, error)
}

type QuizRepository interface {
	Create(ctx context.Context, s *entities.QuizSession) (int64, error)
	GetByID(ctx context.Context, id int64) (*entities.QuizSession, error)
	Update(ctx context.Context, s *entities.QuizSession) error
	SaveAnswer(ctx context.Context, a *entities.QuizAnswer) error
}

type SettingsRepository interface {
	Create(ctx context.Context, userID int64) error
	GetByUserID(ctx context.Context, userID int64) (*entities.UserSettings, error)
	UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error
	UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error
}

// ReminderRepository manages reminder persistence.
type ReminderRepository interface {
	GetDueReminders(ctx context.Context) ([]*entities.ReminderWithUser, error)
	MarkAsSent(ctx context.Context, userID int64, sentAt time.Time) error
	GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error)
	Upsert(ctx context.Context, rem *entities.UserReminders) error
}

// ReminderNotifier sends reminder notifications to users.
type ReminderNotifier interface {
	SendReminder(chatID int64, payload entities.ReminderPayload) error
}
