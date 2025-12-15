package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleSettingsCommand(ctx context.Context, chatID, userID int64) {
	msg := newHTMLMessage(chatID, "")

	settings, err := h.settingsService.GetOrCreate(ctx, userID)
	if err != nil {
		msg.Text = msgSettingsUnavailable
		h.send(msg)
		return
	}

	text := fmt.Sprintf(
		"<b>⚙️ Настройки</b>\n\n"+
			"📚 <b>Имён в день:</b> %d\n"+
			"📝 <b>Длина квиза:</b> %d\n"+
			"🎲 <b>Режим квиза:</b> %s\n"+
			"🔤 <b>Транслитерация:</b> %s\n"+
			"🔊 <b>Аудио:</b> %s\n",
		settings.NamesPerDay,
		settings.QuizLength,
		formatQuizMode(settings.QuizMode),
		formatBool(settings.ShowTransliteration),
		formatBool(settings.ShowAudio),
	)

	kb := buildSettingsKeyboard()

	msg.Text = text
	msg.ReplyMarkup = kb
	h.send(msg)
}

func buildSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📚 Имён в день", "settings:names_per_day"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Длина квиза", "settings:quiz_length"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Режим квиза", "settings:quiz_mode"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔤 Транслитерация", "settings:toggle_transliteration"),
			tgbotapi.NewInlineKeyboardButtonData("🔊 Аудио", "settings:toggle_audio"),
		),
	)
}

func formatQuizMode(mode string) string {
	switch mode {
	case "new_only":
		return "Только новые"
	case "review_only":
		return "Только повторение"
	case "mixed":
		return "Смешанный"
	default:
		return mode
	}
}

func formatBool(b bool) string {
	if b {
		return "Включено ✅"
	}
	return "Выключено ❌"
}
