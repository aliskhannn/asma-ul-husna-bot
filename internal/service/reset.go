package service

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

type ResetService struct {
	tr Transactor
}

func NewResetService(
	tr Transactor,
) *ResetService {
	return &ResetService{
		tr: tr,
	}
}

func (s *ResetService) ResetUser(ctx context.Context, userID int64) error {
	return s.tr.WithinTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		resetRepo := repository.NewResetRepository(tx)
		settingsRepo := repository.NewSettingsRepository(tx)
		reminderRepo := repository.NewRemindersRepository(tx)

		if err := settingsRepo.UpsertDefaults(ctx, userID); err != nil {
			return err
		}

		defRem := entities.NewUserReminders(userID)
		if err := reminderRepo.Upsert(ctx, defRem); err != nil {
			return err
		}

		if err := resetRepo.ResetUser(ctx, userID); err != nil {
			return err
		}

		return nil
	})
}
