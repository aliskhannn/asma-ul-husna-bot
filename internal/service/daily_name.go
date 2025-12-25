package service

import "context"

type DailyNameService struct {
	dailyNameRepo DailyNameRepository
}

func NewDailyNameService(dailyNameRepo DailyNameRepository) *DailyNameService {
	return &DailyNameService{
		dailyNameRepo: dailyNameRepo,
	}
}

func (s *DailyNameService) GetTodayNames(ctx context.Context, userID int64) ([]int, error) {
	return s.dailyNameRepo.GetTodayNames(ctx, userID)
}

func (s *DailyNameService) GetTodayNamesCount(ctx context.Context, userID int64) (int, error) {
	return s.dailyNameRepo.GetTodayNamesCount(ctx, userID)
}

func (s *DailyNameService) HasUnfinishedDays(ctx context.Context, userID int64) (bool, error) {
	return s.dailyNameRepo.HasUnfinishedDays(ctx, userID)
}

func (s *DailyNameService) GetOldestUnfinishedName(ctx context.Context, userID int64) (int, error) {
	return s.dailyNameRepo.GetOldestUnfinishedName(ctx, userID)
}

func (s *DailyNameService) AddTodayName(ctx context.Context, userID int64, nameNumber int) error {
	return s.dailyNameRepo.AddTodayName(ctx, userID, nameNumber)
}

func (s *DailyNameService) RemoveTodayName(ctx context.Context, userID int64, nameNumber int) error {
	return s.dailyNameRepo.RemoveTodayName(ctx, userID, nameNumber)
}
