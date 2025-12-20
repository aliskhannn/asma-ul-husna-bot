package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

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

// ReminderService handles reminder business logic.
type ReminderService struct {
	reminderRepo ReminderRepository
	progressRepo ProgressRepository
	settingsRepo SettingsRepository
	nameRepo     NameRepository
	notifier     ReminderNotifier
	logger       *zap.Logger
}

// NewReminderService creates a new reminder service.
func NewReminderService(
	reminderRepo ReminderRepository,
	progressRepo ProgressRepository,
	settingsRepo SettingsRepository,
	nameRepo NameRepository,
	logger *zap.Logger,
) *ReminderService {
	return &ReminderService{
		reminderRepo: reminderRepo,
		progressRepo: progressRepo,
		settingsRepo: settingsRepo,
		nameRepo:     nameRepo,
		notifier:     nil,
		logger:       logger,
	}
}

// SetNotifier sets the notifier (called after handler is created)
func (s *ReminderService) SetNotifier(notifier ReminderNotifier) {
	s.notifier = notifier
}

// Start begins the reminder scheduling loop.
func (s *ReminderService) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	s.logger.Info("reminder service started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("reminder service stopped")
			return
		case <-ticker.C:
			if err := s.sendHourlyReminders(ctx); err != nil {
				s.logger.Error("failed to send hourly reminders", zap.Error(err))
			}
		}
	}
}

// sendHourlyReminders processes and sends all due reminders.
func (s *ReminderService) sendHourlyReminders(ctx context.Context) error {
	now := time.Now().UTC()

	reminders, err := s.reminderRepo.GetDueReminders(ctx)
	if err != nil {
		return fmt.Errorf("get due reminders: %w", err)
	}

	s.logger.Info("processing reminders", zap.Int("count", len(reminders)))

	sent := 0
	for _, rwu := range reminders {
		if err := s.processReminder(ctx, rwu, now); err != nil {
			s.logger.Error("failed to process reminder",
				zap.Int64("user_id", rwu.UserID),
				zap.Error(err))
		} else {
			sent++
		}
	}

	s.logger.Info("reminders processed",
		zap.Int("total", len(reminders)),
		zap.Int("sent", sent))

	return nil
}

// processReminder handles a single reminder.
func (s *ReminderService) processReminder(
	ctx context.Context,
	rwu *entities.ReminderWithUser,
	now time.Time,
) error {
	// 1. Check if we can send now (time window + interval check).
	if !rwu.CanSendNow() {
		s.logger.Debug("reminder not due yet",
			zap.Int64("user_id", rwu.UserID),
			zap.String("reason", "outside time window or interval not elapsed"))
		return nil
	}

	// 2. Build statistics for the message.
	stats, err := s.buildReminderStats(ctx, rwu)
	if err != nil {
		return fmt.Errorf("build reminder stats: %w", err)
	}

	// 3. Select name by priority
	name, err := s.selectNameForReminder(ctx, rwu.UserID, now, stats)
	if err != nil {
		return fmt.Errorf("select name for reminder: %w", err)
	}

	if name == nil {
		s.logger.Debug("no name to send", zap.Int64("user_id", rwu.UserID))
		return nil
	}

	// 4. Send notification via delivery layer.
	if s.notifier == nil {
		s.logger.Error("notifier not set, cannot send reminder")
		return fmt.Errorf("notifier not initialized")
	}

	payload := entities.ReminderPayload{
		Name:  *name,
		Stats: *stats,
	}

	if err := s.notifier.SendReminder(rwu.ChatID, payload); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	// 5. Mark as sent.
	if err := s.reminderRepo.MarkAsSent(ctx, rwu.UserID, now); err != nil {
		return fmt.Errorf("mark as sent: %w", err)
	}

	s.logger.Info("reminder sent successfully", zap.Int64("user_id", rwu.UserID))
	return nil
}

func (s *ReminderService) selectNameForReminder(
	ctx context.Context,
	userID int64,
	now time.Time,
	stats *entities.ReminderStats,
) (*entities.Name, error) {
	// Приоритет 1: Due-имя (SRS повтор)
	if stats.DueToday > 0 {
		nameNumber, err := s.progressRepo.GetNextDueName(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("get next due name: %w", err)
		}
		if nameNumber > 0 {
			name, err := s.nameRepo.GetByNumber(ctx, nameNumber)
			if err != nil {
				return nil, fmt.Errorf("get name by number: %w", err)
			}
			s.logger.Debug("selected due name",
				zap.Int64("user_id", userID),
				zap.Int("name_number", name.Number))
			return name, nil
		}
	}

	// Приоритет 2: Новое имя дня (учитывая NamesPerDay)
	if stats.NotStarted > 0 {
		// Получаем настройки пользователя
		settings, err := s.settingsRepo.GetByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("get user settings: %w", err)
		}

		namesPerDay := 1
		if settings != nil && settings.NamesPerDay > 0 {
			namesPerDay = settings.NamesPerDay
		}

		dateUTC := now.Truncate(24 * time.Hour)

		// Сначала пробуем получить следующее имя из уже созданного "плана дня"
		nameNumber, err := s.progressRepo.GetNextDailyName(ctx, userID, dateUTC)
		if err != nil {
			return nil, fmt.Errorf("get next daily name: %w", err)
		}

		// Если нет следующего — создаём/получаем план на день
		if nameNumber == 0 {
			nameNumber, err = s.progressRepo.GetOrCreateDailyName(ctx, userID, dateUTC, namesPerDay)
			if err != nil {
				return nil, fmt.Errorf("get or create daily name: %w", err)
			}
		}

		if nameNumber > 0 {
			name, err := s.nameRepo.GetByNumber(ctx, nameNumber)
			if err != nil {
				return nil, fmt.Errorf("get name by number: %w", err)
			}
			s.logger.Debug("selected new daily name",
				zap.Int64("user_id", userID),
				zap.Int("name_number", name.Number),
				zap.Int("names_per_day", namesPerDay))
			return name, nil
		}
	}

	// Приоритет 3: Случайное для закрепления
	nameNumber, err := s.progressRepo.GetRandomLearnedName(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get random learned name: %w", err)
	}

	if nameNumber > 0 {
		name, err := s.nameRepo.GetByNumber(ctx, nameNumber)
		if err != nil {
			return nil, fmt.Errorf("get name by number: %w", err)
		}
		s.logger.Debug("selected random learned name",
			zap.Int64("user_id", userID),
			zap.Int("name_number", name.Number))
		return name, nil
	}

	return nil, nil
}

// buildReminderStats collects statistics for reminder message.
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

	return &entities.ReminderStats{
		DueToday:       stats.DueToday,
		Learned:        stats.Learned,
		NotStarted:     stats.NotStarted,
		DaysToComplete: settings.DaysToComplete(stats.Learned),
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

// ToggleReminder enables or disables reminders for a user.
func (s *ReminderService) ToggleReminder(ctx context.Context, userID int64) error {
	reminder, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get reminder: %w", err)
	}

	reminder.IsEnabled = !reminder.IsEnabled
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder toggled",
		zap.Int64("user_id", userID),
		zap.Bool("enabled", reminder.IsEnabled))

	return nil
}

// SetReminderIntervalHours updates the reminder interval hours.
func (s *ReminderService) SetReminderIntervalHours(
	ctx context.Context,
	userID int64,
	intervalHours int,
) error {
	reminder, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get reminder: %w", err)
	}

	reminder.IntervalHours = intervalHours
	reminder.IsEnabled = true // enable when setting interval hours
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder frequency set",
		zap.Int64("user_id", userID),
		zap.Int("interval hours", intervalHours),
	)

	return nil
}

// SetReminderTimeWindow updates the start and end time for reminders.
func (s *ReminderService) SetReminderTimeWindow(
	ctx context.Context,
	userID int64,
	startTime, endTime string, // "HH:MM:SS" format
) error {
	// Validate time format.
	if _, err := time.Parse("15:04:05", startTime); err != nil {
		return fmt.Errorf("invalid start time format: %w", err)
	}
	if _, err := time.Parse("15:04:05", endTime); err != nil {
		return fmt.Errorf("invalid end time format: %w", err)
	}

	// Validate that end time is after start time.
	if endTime <= startTime {
		return fmt.Errorf("end time must be after start time")
	}

	reminder, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get reminder: %w", err)
	}

	reminder.StartTimeUTC = startTime
	reminder.EndTimeUTC = endTime
	reminder.IsEnabled = true // enable when setting time window
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder time window set",
		zap.Int64("user_id", userID),
		zap.String("start_time", startTime),
		zap.String("end_time", endTime))

	return nil
}

// SnoozeReminder postpones the next reminder by marking last sent time.
func (s *ReminderService) SnoozeReminder(ctx context.Context, userID int64, duration time.Duration) error {
	reminder, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get reminder: %w", err)
	}

	// Calculate snooze time (current time - interval hours + snooze duration)
	// This way, next reminder will be sent after duration.
	snoozeTime := time.Now().UTC().Add(duration - time.Duration(reminder.IntervalHours)*time.Hour)
	reminder.LastSentAt = &snoozeTime
	reminder.UpdatedAt = time.Now()

	if err := s.reminderRepo.Upsert(ctx, reminder); err != nil {
		return fmt.Errorf("upsert reminder: %w", err)
	}

	s.logger.Info("reminder snoozed",
		zap.Int64("user_id", userID),
		zap.Duration("duration", duration))

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
