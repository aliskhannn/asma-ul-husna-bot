package service

import (
	"context"
	"time"
)

type DailyNameService struct {
	dailyNameRepo DailyNameRepository
	progressRepo  ProgressRepository
}

func NewDailyNameService(dailyNameRepo DailyNameRepository, progressRepo ProgressRepository) *DailyNameService {
	return &DailyNameService{
		dailyNameRepo: dailyNameRepo,
		progressRepo:  progressRepo,
	}
}

// localMidnightToUTCDate returns UTC date representing user's local day start.
func localMidnightToUTCDate(tz string, now time.Time) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	n := now.In(loc)
	localMidnight := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	return localMidnight.UTC().Truncate(24 * time.Hour)
}

func (s *DailyNameService) EnsureTodayPlan(ctx context.Context, userID int64, tz string, namesPerDay int) error {
	if namesPerDay <= 0 {
		namesPerDay = 1
	}

	todayDateUTC := localMidnightToUTCDate(tz, time.Now())

	planned, err := s.dailyNameRepo.GetNamesByDate(ctx, userID, todayDateUTC)
	if err != nil {
		return err
	}

	plannedSet := make(map[int]struct{}, len(planned))
	for _, n := range planned {
		plannedSet[n] = struct{}{}
	}

	remaining := namesPerDay - len(planned)
	if remaining <= 0 {
		return nil
	}

	debt, err := s.dailyNameRepo.GetCarryOverUnfinishedFromPast(ctx, userID, todayDateUTC, remaining)
	if err != nil {
		return err
	}
	for _, n := range debt {
		if _, exists := plannedSet[n]; exists {
			continue
		}
		if err := s.dailyNameRepo.AddNameForDate(ctx, userID, todayDateUTC, n); err != nil {
			return err
		}
		plannedSet[n] = struct{}{}
		remaining--
		if remaining == 0 {
			return nil
		}
	}

	for remaining > 0 {
		newNums, err := s.progressRepo.GetNamesForIntroduction(ctx, userID, remaining)
		if err != nil {
			return err
		}
		if len(newNums) == 0 {
			return nil
		}

		added := 0
		for _, n := range newNums {
			if _, exists := plannedSet[n]; exists {
				continue
			}
			if err := s.dailyNameRepo.AddNameForDate(ctx, userID, todayDateUTC, n); err != nil {
				return err
			}
			plannedSet[n] = struct{}{}
			added++
			remaining--
			if remaining == 0 {
				return nil
			}
		}

		if added == 0 {
			return nil
		}
	}

	return nil
}

func (s *DailyNameService) GetTodayNamesTZ(ctx context.Context, userID int64, tz string) ([]int, error) {
	todayDateUTC := localMidnightToUTCDate(tz, time.Now())
	return s.dailyNameRepo.GetNamesByDate(ctx, userID, todayDateUTC)
}

func (s *DailyNameService) AddTodayNameTZ(ctx context.Context, userID int64, tz string, nameNumber int) error {
	todayDateUTC := localMidnightToUTCDate(tz, time.Now())
	return s.dailyNameRepo.AddNameForDate(ctx, userID, todayDateUTC, nameNumber)
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
