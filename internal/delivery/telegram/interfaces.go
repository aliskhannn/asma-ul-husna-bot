package telegram

import (
	"context"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

type UserService interface {
	EnsureUser(ctx context.Context, userID int64, firstName, lastName string, username string, languageCode string) error
}

type NameService interface {
	GetNameByNumber(ctx context.Context, number int) (entities.Name, error)
	GetRandomName(ctx context.Context) (entities.Name, error)
	GetAllNames(ctx context.Context) ([]entities.Name, error)
}

type ProgressService interface {
	GetProgressSummary(ctx context.Context, userID int64, namesPerDay int) (*service.ProgressSummary, error)
}

type SettingsService interface {
	GetOrCreate(ctx context.Context, userID int64) (*entities.UserSettings, error)
}
