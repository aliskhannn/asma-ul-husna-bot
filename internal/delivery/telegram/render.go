package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// RenderProgress renders a progress message with an optional keyboard.
func (h *Handler) RenderProgress(ctx context.Context, userID int64, withKeyboard bool) (string, *tgbotapi.InlineKeyboardMarkup, error) {
	summary, err := h.progressService.GetProgressSummary(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get progress summary",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return "", nil, err
	}

	progressBar := buildProgressBar(summary.Learned, 99, 20)
	text := formatProgressMessage(summary, progressBar)

	var keyboard *tgbotapi.InlineKeyboardMarkup
	if withKeyboard {
		kb := buildProgressKeyboard()
		keyboard = &kb
	}

	return text, keyboard, nil
}

// RenderSettings renders a settings message with a keyboard.
func (h *Handler) RenderSettings(ctx context.Context, userID int64) (string, tgbotapi.InlineKeyboardMarkup, error) {
	settings, err := h.settingsService.GetOrCreate(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get settings",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	reminders, err := h.reminderService.GetOrCreate(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get or create reminders",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	reminderStatus := formatReminderStatus(reminders)
	learningModeText := formatLearningMode(entities.LearningMode(settings.LearningMode))
	quizMode := formatQuizMode(settings.QuizMode)

	text := fmt.Sprintf(
		"%s\n\n%s\n%s\n%s\n%s",
		md("⚙️ Настройки"),
		md(fmt.Sprintf("🎯 Режим обучения: %s", learningModeText)),
		md(fmt.Sprintf("📚 Имён в день: %d", settings.NamesPerDay)),
		md(fmt.Sprintf("🎲 Режим квиза: %s", quizMode)),
		md(fmt.Sprintf("⏰ Напоминания: %s", reminderStatus)),
	)

	kb := buildSettingsKeyboard()
	return text, kb, nil
}
