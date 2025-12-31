package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres/repository"
)

// ReminderService handles reminder business logic with batch processing.
type ReminderService struct {
	reminderRepo  ReminderRepository
	progressRepo  ProgressRepository
	settingsRepo  SettingsRepository
	nameRepo      NameRepository
	dailyNameRepo DailyNameRepository
	notifier      ReminderNotifier
	logger        *zap.Logger
}

// NewReminderService creates a new reminder service.
func NewReminderService(
	reminderRepo ReminderRepository,
	progressRepo ProgressRepository,
	settingsRepo SettingsRepository,
	nameRepo NameRepository,
	dailyNameRepo DailyNameRepository,
	logger *zap.Logger,
) *ReminderService {
	return &ReminderService{
		reminderRepo:  reminderRepo,
		progressRepo:  progressRepo,
		settingsRepo:  settingsRepo,
		nameRepo:      nameRepo,
		dailyNameRepo: dailyNameRepo,
		logger:        logger,
	}
}

// SetNotifier sets the notifier (called after handler is created).
func (s *ReminderService) SetNotifier(notifier ReminderNotifier) {
	s.notifier = notifier
}

// Start begins the reminder scheduling loop.
func (s *ReminderService) Start(ctx context.Context) {
	s.logger.Info("reminder service started")

	c := cron.New(cron.WithLocation(time.UTC))

	_, err := c.AddFunc("0 * * * *", func() {
		s.logger.Info("cron triggered: processing hourly reminders")
		if err := s.sendHourlyReminders(ctx); err != nil {
			s.logger.Error("failed to send hourly reminders", zap.Error(err))
		}
	})
	if err != nil {
		s.logger.Error("failed to add cron job", zap.Error(err))
		return
	}

	c.Start()
	s.logger.Info("cron scheduler started")

	<-ctx.Done()

	c.Stop()
	s.logger.Info("reminder service stopped")
}

// sendHourlyReminders processes and sends all due reminders in batches.
func (s *ReminderService) sendHourlyReminders(ctx context.Context) error {
	const batchSize = 100
	offset := 0
	totalSent := 0
	now := time.Now().UTC()

	s.logger.Info("processing hourly reminders", zap.Time("now", now))

	for {
		// Fetch reminders in batches
		reminders, err := s.reminderRepo.GetDueRemindersBatch(ctx, now, batchSize, offset)
		if err != nil {
			return fmt.Errorf("get due reminders batch: %w", err)
		}

		if len(reminders) == 0 {
			break // No more reminders
		}

		// Process batch concurrently with rate limiting
		sent := s.processBatch(ctx, reminders)
		totalSent += sent

		if len(reminders) < batchSize {
			break // Last batch
		}

		offset += batchSize
	}

	s.logger.Info("reminders processed",
		zap.Int("total_sent", totalSent),
	)

	return nil
}

// processBatch processes a batch of reminders concurrently.
func (s *ReminderService) processBatch(ctx context.Context, reminders []*entities.ReminderWithUser) int {
	const maxConcurrent = 10
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sent := 0
	now := time.Now().UTC()

	for _, rwu := range reminders {
		wg.Add(1)
		sem <- struct{}{} // Acquire

		go func() {
			defer wg.Done()
			defer func() { <-sem }() // Release

			if err := s.processReminder(ctx, rwu, now); err != nil {
				s.logger.Error("failed to process reminder",
					zap.Int64("user_id", rwu.UserID),
					zap.Error(err))
			} else {
				mu.Lock()
				sent++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return sent
}

// processReminder handles a single reminder.
func (s *ReminderService) processReminder(
	ctx context.Context,
	rwu *entities.ReminderWithUser,
	now time.Time,
) error {
	// 1. Check if we can send now (time window + interval check)
	if !rwu.CanSendNow(now) {
		s.logger.Debug("reminder not due yet",
			zap.Int64("user_id", rwu.UserID),
			zap.String("reason", "outside time window or interval not elapsed"),
		)
		return nil
	}

	// 2. Build statistics for the message
	stats, err := s.buildReminderStats(ctx, rwu)
	if err != nil {
		return fmt.Errorf("build reminder stats: %w", err)
	}

	// 3. Select name by priority
	name, kind, err := s.selectNameForReminder(ctx, rwu.UserID, stats, rwu.LastKind)
	if err != nil {
		return fmt.Errorf("select name for reminder: %w", err)
	}

	if name == nil {
		s.logger.Debug("no name to send", zap.Int64("user_id", rwu.UserID))

		nextSendAt := nextHourUTC(now)

		if err := s.reminderRepo.RescheduleNext(ctx, rwu.UserID, nextSendAt); err != nil {
			return fmt.Errorf("reschedule next send: %w", err)
		}
		return nil
	}

	// 4. Send notification via delivery layer
	if s.notifier == nil {
		s.logger.Error("notifier not set, cannot send reminder")
		return fmt.Errorf("notifier not initialized")
	}

	payload := &entities.ReminderPayload{
		Kind:  kind,
		Name:  *name,
		Stats: *stats,
	}

	if err := s.notifier.SendReminder(rwu.UserID, rwu.ChatID, *payload); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	// 5. Calculate next send time and update
	reminder := &entities.UserReminders{
		UserID:        rwu.UserID,
		IntervalHours: rwu.IntervalHours,
		StartTime:     rwu.StartTime,
		EndTime:       rwu.EndTime,
	}
	nextSendAt := reminder.CalculateNextSendAt(rwu.Timezone, now)

	nextLastKind := nextKindForAlternation(rwu.LastKind, kind)

	if err := s.reminderRepo.UpdateAfterSend(ctx, rwu.UserID, now, nextSendAt, nextLastKind); err != nil {
		return fmt.Errorf("update after send: %w", err)
	}

	s.logger.Info("reminder sent successfully",
		zap.Int64("user_id", rwu.UserID),
		zap.Int("name_number", name.Number),
		zap.Time("next_send_at", nextSendAt),
	)

	return nil
}

func nextHourUTC(t time.Time) time.Time {
	tt := t.UTC().Truncate(time.Hour).Add(time.Hour)
	return tt
}

// selectNameForReminder selects a name to send based on priority.
// selectNameForReminder selects a name to send based on priority.
func (s *ReminderService) selectNameForReminder(
	ctx context.Context,
	userID int64,
	stats *entities.ReminderStats,
	last entities.ReminderKind,
) (*entities.Name, entities.ReminderKind, error) {
	prefer := preferredKind(last)

	// Load settings first to get timezone and namesPerDay.
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("get user settings: %w", err)
	}

	tz := "UTC"
	namesPerDay := 1
	learningMode := string(entities.ModeGuided)

	if settings != nil {
		if settings.Timezone != "" {
			tz = settings.Timezone
		}
		if settings.NamesPerDay > 0 {
			namesPerDay = settings.NamesPerDay
		}
		if settings.LearningMode != "" {
			learningMode = settings.LearningMode
		}
	}

	// Ensure today's plan exists before selecting from it.
	// Guided mode respects "debt first" via the plan-filling logic.
	if namesPerDay <= 0 {
		namesPerDay = 1
	}
	todayDateUTC := localMidnightToUTCDate(tz, time.Now())

	planned, err := s.dailyNameRepo.GetNamesByDate(ctx, userID, todayDateUTC)
	if err != nil {
		return nil, "", fmt.Errorf("get names by date: %w", err)
	}

	plannedSet := make(map[int]struct{}, len(planned))
	for _, n := range planned {
		plannedSet[n] = struct{}{}
	}

	remaining := namesPerDay - len(planned)
	if remaining > 0 {
		// Carry over learning names from previous plans first.
		if learningMode == string(entities.ModeGuided) {
			debt, err := s.dailyNameRepo.GetCarryOverUnfinishedFromPast(ctx, userID, todayDateUTC, remaining)
			if err != nil {
				return nil, "", fmt.Errorf("get carry over learning: %w", err)
			}
			for _, n := range debt {
				if _, exists := plannedSet[n]; exists {
					continue
				}
				if err := s.dailyNameRepo.AddNameForDate(ctx, userID, todayDateUTC, n); err != nil {
					return nil, "", fmt.Errorf("add name for date: %w", err)
				}
				plannedSet[n] = struct{}{}
				remaining--
				if remaining == 0 {
					break
				}
			}
		}

		// Fill the rest with not-yet-introduced names.
		for remaining > 0 {
			newNums, err := s.progressRepo.GetNamesForIntroduction(ctx, userID, remaining)
			if err != nil {
				return nil, "", fmt.Errorf("get names for introduction: %w", err)
			}
			if len(newNums) == 0 {
				break
			}

			added := 0
			for _, n := range newNums {
				if _, exists := plannedSet[n]; exists {
					continue
				}
				if err := s.dailyNameRepo.AddNameForDate(ctx, userID, todayDateUTC, n); err != nil {
					return nil, "", fmt.Errorf("add name for date: %w", err)
				}
				plannedSet[n] = struct{}{}
				added++
				remaining--
				if remaining == 0 {
					break
				}
			}
			if added == 0 {
				break
			}
		}
	}

	// Priority 1: Due names (SRS).
	var reviewName *entities.Name
	if stats != nil && stats.DueToday > 0 {
		nameNumber, err := s.progressRepo.GetNextDueName(ctx, userID)
		if err != nil {
			return nil, "", fmt.Errorf("get next due name: %w", err)
		}
		if nameNumber > 0 {
			name, err := s.nameRepo.GetByNumber(nameNumber)
			if err != nil {
				return nil, "", fmt.Errorf("get name by number: %w", err)
			}
			reviewName = name
		}
	}

	// Priority 2: Today's names (plan-based), but only not-mastered.
	var studyName *entities.Name
	todayNames, err := s.dailyNameRepo.GetTodayNames(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("get today names: %w", err)
	}

	candidates := make([]int, 0, len(todayNames))
	for _, n := range todayNames {
		streak, err := s.progressRepo.GetStreak(ctx, userID, n)
		if err != nil {
			// No progress means not mastered yet.
			candidates = append(candidates, n)
			continue
		}
		if streak < entities.MinStreakForMastery {
			candidates = append(candidates, n)
		}
	}

	if len(candidates) > 0 {
		nameNumber := candidates[rand.Intn(len(candidates))]
		name, err := s.nameRepo.GetByNumber(nameNumber)
		if err != nil {
			return nil, "", fmt.Errorf("get name by number: %w", err)
		}
		studyName = name
	}

	// "New" is defined as a planned name that has no progress record yet.
	// This keeps ReminderService read-only and makes "new" depend on the daily plan.
	var newName *entities.Name
	for _, n := range todayNames {
		_, err := s.progressRepo.Get(ctx, userID, n)
		if err == nil {
			continue
		}
		// Treat not found as "new"; other errors should be returned.
		if !errors.Is(err, repository.ErrProgressNotFound) {
			return nil, "", fmt.Errorf("get progress: %w", err)
		}

		nm, err := s.nameRepo.GetByNumber(n)
		if err != nil {
			return nil, "", fmt.Errorf("get name by number: %w", err)
		}
		newName = nm
		break
	}

	// prefer NEW
	if prefer == entities.ReminderKindNew {
		if newName != nil {
			return newName, entities.ReminderKindNew, nil
		}
		if reviewName != nil {
			return reviewName, entities.ReminderKindReview, nil
		}
		if studyName != nil {
			return studyName, entities.ReminderKindStudy, nil
		}
		return nil, "", nil
	}

	// prefer REVIEW
	if reviewName != nil {
		return reviewName, entities.ReminderKindReview, nil
	}
	if newName != nil {
		return newName, entities.ReminderKindNew, nil
	}
	if studyName != nil {
		return studyName, entities.ReminderKindStudy, nil
	}

	return nil, "", nil
}

func nextKindForAlternation(prev entities.ReminderKind, sent entities.ReminderKind) entities.ReminderKind {
	if sent == entities.ReminderKindStudy {
		if prev == "" {
			return entities.ReminderKindNew
		}
		return prev
	}
	// если отправили new/review — запоминаем его
	if sent == entities.ReminderKindNew || sent == entities.ReminderKindReview {
		return sent
	}
	// safety
	if prev == "" {
		return entities.ReminderKindNew
	}
	return prev
}

func preferredKind(prev entities.ReminderKind) entities.ReminderKind {
	if prev == entities.ReminderKindNew {
		return entities.ReminderKindReview
	}

	return entities.ReminderKindNew
}

// buildReminderStats collects statistics for the reminder message.
func (s *ReminderService) buildReminderStats(
	ctx context.Context,
	rem *entities.ReminderWithUser,
) (*entities.ReminderStats, error) {
	stats, err := s.progressRepo.GetStats(ctx, rem.UserID)
	if err != nil {
		return nil, fmt.Errorf("get progress stats: %w", err)
	}

	settings, err := s.settingsRepo.GetByUserID(ctx, rem.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user settings: %w", err)
	}

	daysToComplete := 0
	if settings != nil {
		daysToComplete = settings.DaysToComplete(stats.Learned)
	}

	return &entities.ReminderStats{
		DueToday:       stats.DueToday,
		Learned:        stats.Learned,
		NotStarted:     stats.NotStarted,
		DaysToComplete: daysToComplete,
	}, nil
}

// GetByUserID retrieves reminder settings for a user.
func (s *ReminderService) GetByUserID(ctx context.Context, userID int64) (*entities.UserReminders, error) {
	reminder, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get reminder: %w", err)
	}

	// If no reminder exists, create default.
	if reminder == nil {
		reminder = entities.NewUserReminders(userID)
		if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
			return nil, fmt.Errorf("create default reminder: %w", err)
		}
	}

	return reminder, nil
}

// GetOrCreate retrieves reminder settings or creates default ones.
func (s *ReminderService) GetOrCreate(ctx context.Context, userID int64) (*entities.UserReminders, error) {
	reminder, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrReminderNotFound) {
			// Create default reminder settings
			reminder = entities.NewUserReminders(userID)
			if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
				return nil, fmt.Errorf("create default reminder: %w", err)
			}
			return reminder, nil
		}
		return nil, fmt.Errorf("get reminder: %w", err)
	}

	return reminder, nil
}

// ToggleReminder enables or disables reminders for a user.
func (s *ReminderService) ToggleReminder(ctx context.Context, userID int64) error {
	reminder, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		// Create default if not found
		if errors.Is(err, repository.ErrReminderNotFound) {
			reminder = entities.NewUserReminders(userID)
		} else {
			return fmt.Errorf("get reminder: %w", err)
		}
	}

	reminder.IsEnabled = !reminder.IsEnabled
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder toggled",
		zap.Int64("user_id", userID),
		zap.Bool("enabled", reminder.IsEnabled),
	)

	return nil
}

// SnoozeReminder postpones the next reminder to the next scheduler tick after the given duration.
// The tick is aligned to the user's configured reminder interval (e.g., every 2h/4h/6h).
// SnoozeReminder postpones the next reminder to the next full UTC hour.
// Works with the hourly cron dispatcher.
func (s *ReminderService) SnoozeReminder(ctx context.Context, userID int64) error {
	reminder, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get reminder: %w", err)
	}

	nowUTC := time.Now().UTC()
	next := nowUTC.Truncate(time.Hour).Add(time.Hour)

	reminder.IsEnabled = true
	reminder.NextSendAt = &next
	reminder.UpdatedAt = nowUTC

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}
	return nil
}

// DisableReminder disables reminders for a user.
func (s *ReminderService) DisableReminder(ctx context.Context, userID int64) error {
	reminder, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get reminder: %w", err)
	}

	reminder.IsEnabled = false
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder disabled", zap.Int64("user_id", userID))

	return nil
}

// SetReminderIntervalHours updates the reminder interval hours.
func (s *ReminderService) SetReminderIntervalHours(ctx context.Context, userID int64, intervalHours int) error {
	reminder, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrReminderNotFound) {
			reminder = entities.NewUserReminders(userID)
		} else {
			return fmt.Errorf("get reminder: %w", err)
		}
	}

	tz := "UTC"
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err == nil && settings != nil && settings.Timezone != "" {
		tz = settings.Timezone
	}

	if intervalHours <= 0 {
		intervalHours = 1
	}

	reminder.IntervalHours = intervalHours
	reminder.IsEnabled = true
	reminder.UpdatedAt = time.Now().UTC()

	// Recalculate next_send_at because interval changed
	next := reminder.CalculateNextSendAt(tz, time.Now().UTC())
	reminder.NextSendAt = &next

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder frequency set",
		zap.Int64("user_id", userID),
		zap.Int("interval_hours", intervalHours),
		zap.String("timezone", tz),
		zap.Time("next_send_at", next),
	)

	return nil
}

// SetReminderTimeWindow updates the start and end time for reminders.
func (s *ReminderService) SetReminderTimeWindow(
	ctx context.Context,
	userID int64,
	startTime, endTime string,
) error {
	reminder, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrReminderNotFound) {
			reminder = entities.NewUserReminders(userID)
		} else {
			return fmt.Errorf("get reminder: %w", err)
		}
	}

	tz := "UTC"
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err == nil && settings != nil && settings.Timezone != "" {
		tz = settings.Timezone
	}

	// Validate the configured daily time window. The values are treated as a "time of day"
	// and must follow the "HH:MM:SS" format.
	startTOD, err := time.Parse("15:04:05", startTime)
	if err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}
	endTOD, err := time.Parse("15:04:05", endTime)
	if err != nil {
		return fmt.Errorf("invalid end time: %w", err)
	}
	if !endTOD.After(startTOD) {
		return fmt.Errorf("invalid time window: endTime must be after startTime")
	}

	nowUTC := time.Now().UTC()

	reminder.StartTime = startTime
	reminder.EndTime = endTime
	reminder.IsEnabled = true
	reminder.UpdatedAt = nowUTC

	// Recalculate the next send time immediately so the scheduler can pick it up right away.
	next := reminder.CalculateNextSendAt(tz, nowUTC)
	reminder.NextSendAt = &next

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder time window set",
		zap.Int64("user_id", userID),
		zap.String("timezone", tz),
		zap.String("start_time", startTime),
		zap.String("end_time", endTime),
		zap.Time("next_send_at", next),
	)

	return nil
}
