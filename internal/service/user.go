package service

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres/repository"
)

// UserService provides business logic for user management.
type UserService struct {
	tr           Transactor
	userRepo     UserRepository
	settingsRepo SettingsRepository
}

// NewUserService creates a new UserService with the provided repository.
func NewUserService(
	tr Transactor,
	userRepo UserRepository,
) *UserService {
	return &UserService{
		tr:       tr,
		userRepo: userRepo,
	}
}

// EnsureUser checks if a user exists and creates one if not.
// It does nothing if the user already exists.
func (s *UserService) EnsureUser(ctx context.Context, userID, chatID int64) (bool, error) {
	user := entities.NewUser(userID, chatID)

	var created bool
	err := s.tr.WithinTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		userRepoTx := repository.NewUserRepository(tx)
		settingsRepoTx := repository.NewSettingsRepository(tx)

		c, err := userRepoTx.Save(ctx, user)
		if err != nil {
			return err
		}
		created = c

		return settingsRepoTx.Create(ctx, userID)
	})

	return created, err
}

func (s *UserService) Exists(ctx context.Context, userID int64) (bool, error) {
	return s.userRepo.Exists(ctx, userID)
}
