package service

import "context"

type ResetService struct {
	repo ResetRepository
}

func NewResetService(r ResetRepository) *ResetService {
	return &ResetService{repo: r}
}

func (s *ResetService) ResetUser(ctx context.Context, userID int64) error {
	return s.repo.ResetUser(ctx, userID)
}
