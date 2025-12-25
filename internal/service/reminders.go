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
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
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

		go func(r *entities.ReminderWithUser) {
			defer wg.Done()
			defer func() { <-sem }() // Release

			if err := s.processReminder(ctx, r, now); err != nil {
				s.logger.Error("failed to process reminder",
					zap.Int64("user_id", r.UserID),
					zap.Error(err))
			} else {
				mu.Lock()
				sent++
				mu.Unlock()
			}
		}(rwu)
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
	name, kind, err := s.selectNameForReminder(ctx, rwu.UserID, now, stats, rwu.LastKind)
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

	if err := s.notifier.SendReminder(rwu.ChatID, *payload); err != nil {
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
func (s *ReminderService) selectNameForReminder(
	ctx context.Context,
	userID int64,
	now time.Time,
	stats *entities.ReminderStats,
	last entities.ReminderKind,
) (*entities.Name, entities.ReminderKind, error) {
	prefer := preferredKind(last)

	// Priority 1: Due names (SRS)
	var reviewName *entities.Name
	if stats.DueToday > 0 {
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

	// Priority 2: TODAY's names (user_daily_name) - repeat current names
	var studyName *entities.Name
	todayNames, err := s.dailyNameRepo.GetTodayNames(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("get today names: %w", err)
	}
	if len(todayNames) > 0 {
		nameNumber := todayNames[rand.Intn(len(todayNames))]
		name, err := s.nameRepo.GetByNumber(nameNumber)
		if err != nil {
			return nil, "", fmt.Errorf("get name by number: %w", err)
		}
		studyName = name
	}

	// Priority 3: NEW - Introduction based on namesPerDay quota (user_daily_name)
	var newName *entities.Name
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("get user settings: %w", err)
	}
	namesPerDay := 1
	if settings != nil && settings.NamesPerDay > 0 {
		namesPerDay = settings.NamesPerDay
	}

	allowNew := true
	if settings != nil && settings.LearningMode == string(entities.ModeGuided) {
		hasDebt, err := s.dailyNameRepo.HasUnfinishedDays(ctx, userID)
		if err != nil {
			return nil, "", fmt.Errorf("has unfinished days: %w", err)
		}
		if hasDebt {
			allowNew = false
		}
	}

	if allowNew {
		todayCount, err := s.dailyNameRepo.GetTodayNamesCount(ctx, userID)
		if err != nil {
			return nil, "", fmt.Errorf("get today names count: %w", err)
		}

		if todayCount < namesPerDay {
			nums, err := s.progressRepo.GetNamesForIntroduction(ctx, userID, 1)
			if err != nil {
				return nil, "", fmt.Errorf("get names for introduction: %w", err)
			}
			if len(nums) > 0 {
				n := nums[0]
				if err := s.progressRepo.MarkAsIntroduced(ctx, userID, n); err != nil {
					return nil, "", fmt.Errorf("mark as introduced: %w", err)
				}
				if err := s.dailyNameRepo.AddTodayName(ctx, userID, n); err != nil {
					return nil, "", fmt.Errorf("add today name: %w", err)
				}
				nm, err := s.nameRepo.GetByNumber(n)
				if err != nil {
					return nil, "", fmt.Errorf("get name by number: %w", err)
				}
				newName = nm
			}
		}
	}

	// Priority 4: Random learned name for reinforcement
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

// SnoozeReminder postpones the next reminder by marking last sent time.
func (s *ReminderService) SnoozeReminder(ctx context.Context, userID int64, duration time.Duration) error {
	reminder, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get reminder: %w", err)
	}

	// nearest cron tick after duration (hour-aligned)
	now := time.Now().UTC()
	target := now.Add(duration)

	next := target.Truncate(time.Hour)
	if next.Before(target) {
		next = next.Add(time.Hour)
	}

	reminder.NextSendAt = &next
	reminder.UpdatedAt = time.Now().UTC()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}
	return nil
}

// TODO: удалить, если не используется. Есть уже ToggleReminder

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
		if err == repository.ErrReminderNotFound {
			reminder = entities.NewUserReminders(userID)
		} else {
			return fmt.Errorf("get reminder: %w", err)
		}
	}

	reminder.IntervalHours = intervalHours
	reminder.IsEnabled = true // Enable when setting interval
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder frequency set",
		zap.Int64("user_id", userID),
		zap.Int("interval_hours", intervalHours),
	)

	return nil
}

// SetReminderTimeWindow updates the start and end time for reminders.
func (s *ReminderService) SetReminderTimeWindow(
	ctx context.Context,
	userID int64,
	startTime, endTime string,
) error {
	// Validate time format (HH:MM:SS)
	if _, err := time.Parse("15:04:05", startTime); err != nil {
		return fmt.Errorf("invalid start time format: %w", err)
	}
	if _, err := time.Parse("15:04:05", endTime); err != nil {
		return fmt.Errorf("invalid end time format: %w", err)
	}

	// Validate that end time is after start time
	if endTime <= startTime {
		return fmt.Errorf("end time must be after start time")
	}

	reminder, err := s.reminderRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == repository.ErrReminderNotFound {
			reminder = entities.NewUserReminders(userID)
		} else {
			return fmt.Errorf("get reminder: %w", err)
		}
	}

	reminder.StartTime = startTime
	reminder.EndTime = endTime
	reminder.IsEnabled = true // Enable when setting time window
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder time window set",
		zap.Int64("user_id", userID),
		zap.String("start_time", startTime),
		zap.String("end_time", endTime),
	)

	return nil
}
