package telegram

import (
	"context"
	"fmt"
	"strings"
)

func (h *Handler) handleProgressCommand(ctx context.Context, userID int64) {
	msg := newHTMLMessage(userID, "")

	settings, err := h.settingsService.GetOrCreate(ctx, userID)
	if err != nil {
		msg.Text = msgSettingsUnavailable
		h.send(msg)
		return
	}

	summary, err := h.progressService.GetProgressSummary(ctx, userID, settings.NamesPerDay)
	if err != nil {
		msg.Text = msgProgressUnavailable
		h.send(msg)
		return
	}

	progressBar := buildProgressBar(summary.Learned, 99, 20)

	text := fmt.Sprintf(
		"<b>📊 Ваш прогресс</b>\n\n"+
			"%s\n\n"+
			"✅ <b>Выучено:</b> %d / 99 (%.1f%%)\n"+
			"📖 <b>В процессе:</b> %d\n"+
			"⏳ <b>Не начато:</b> %d\n\n"+
			"🎯 <b>Точность:</b> %.1f%%\n"+
			"📅 <b>Имён в день:</b> %d\n"+
			"⏰ <b>Дней до завершения:</b> %d\n",
		progressBar,
		summary.Learned,
		summary.Percentage,
		summary.InProgress,
		summary.NotStarted,
		summary.Accuracy,
		settings.NamesPerDay,
		summary.DaysToComplete,
	)

	msg.Text = text
	h.send(msg)
}

// buildProgressBar creates ASCII progress bar.
func buildProgressBar(current, total, length int) string {
	if total == 0 {
		return strings.Repeat("░", length)
	}

	filled := int(float64(current) / float64(total) * float64(length))
	if filled > length {
		filled = length
	}

	empty := length - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("[%s]", bar)
}
