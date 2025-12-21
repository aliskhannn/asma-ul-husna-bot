// render.go contains functions for rendering Telegram messages and keyboards.

package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RenderProgress renders a progress message with an optional keyboard.
func (h *Handler) RenderProgress(ctx context.Context, userID int64, withKeyboard bool) (string, *tgbotapi.InlineKeyboardMarkup, error) {
	settings, err := h.settingsService.GetOrCreate(ctx, userID)
	if err != nil {
		return "", nil, err
	}

	summary, err := h.progressService.GetProgressSummary(ctx, userID, settings.NamesPerDay)
	if err != nil {
		return "", nil, err
	}

	progressBar := buildProgressBar(summary.Learned, 99, 20)

	text := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n%s\n%s\n\n%s\n%s\n%s\n",
		md("📊 Ваш прогресс"),
		md(progressBar),
		md(fmt.Sprintf("✅ Выучено: %d / 99 (%.1f%%)", summary.Learned, summary.Percentage)),
		md(fmt.Sprintf("📖 В процессе: %d", summary.InProgress)),
		md(fmt.Sprintf("⏳ Не начато: %d", summary.NotStarted)),
		md(fmt.Sprintf("🎯 Точность: %.1f%%", summary.Accuracy)),
		md(fmt.Sprintf("📅 Имён в день: %d", settings.NamesPerDay)),
		md(fmt.Sprintf("⏰ Дней до завершения: %d", summary.DaysToComplete)),
	)

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
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	reminders, err := h.reminderService.GetByUserID(ctx, userID)
	if err != nil {
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	reminderStatus := formatReminderStatus(reminders)

	text := fmt.Sprintf(
		"%s\n\n%s\n%s\n%s",
		md("⚙️ Настройки"),
		md(fmt.Sprintf("📚 Имён в день: %d", settings.NamesPerDay)),
		md(fmt.Sprintf("🎲 Режим квиза: %s", formatQuizMode(settings.QuizMode))),
		md(fmt.Sprintf("⏰ Напоминания: %s", reminderStatus)),
	)

	kb := buildSettingsKeyboard()
	return text, kb, nil
}
