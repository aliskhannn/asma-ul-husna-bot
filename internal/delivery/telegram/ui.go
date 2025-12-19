package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// buildNameKeyboard builds pagination keyboard for names list.
func buildNameKeyboard(page, totalPages int, prevData, nextData string) *tgbotapi.InlineKeyboardMarkup {
	if totalPages <= 1 {
		return nil
	}

	var row []tgbotapi.InlineKeyboardButton
	if page > 0 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("◀️ Предыдущее", prevData))
	}

	if page < totalPages-1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("Следующее ▶️", nextData))
	}

	kb := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{row},
	}

	return &kb
}

// buildProgressKeyboard builds keyboard for progress screen.
func buildProgressKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", buildProgressCallback()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎯 Начать квиз", buildQuizStartCallback()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", buildSettingsCallback(settingsMenu)),
		),
	)
}

// buildSettingsKeyboard builds main settings keyboard.
func buildSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📚 Имён в день", buildSettingsCallback(settingsNamesPerDay)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Режим квиза", buildSettingsCallback(settingsQuizMode)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Мой прогресс", buildProgressCallback()),
		),
	)
}

// buildQuizResultKeyboard builds keyboard for quiz results screen.
func buildQuizResultKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Новый квиз", buildQuizStartCallback()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Мой прогресс", buildProgressCallback()),
		),
	)
}

// buildQuizAnswerKeyboard builds keyboard for quiz question.
func buildQuizAnswerKeyboard(q *entities.Question, sessionID int64, questionNum int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, option := range q.Options {
		callbackData := buildQuizAnswerCallback(sessionID, questionNum, i)
		button := tgbotapi.NewInlineKeyboardButtonData(option, callbackData)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// buildNamesPerDayKeyboard builds keyboard for names per day setting.
func buildNamesPerDayKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1️⃣ (99 дней)", buildSettingsCallback(settingsNamesPerDay, "1")),
			tgbotapi.NewInlineKeyboardButtonData("2️⃣ (50 дней)", buildSettingsCallback(settingsNamesPerDay, "2")),
			tgbotapi.NewInlineKeyboardButtonData("3️⃣ (33 дня)", buildSettingsCallback(settingsNamesPerDay, "3")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("5️⃣ (20 дней)", buildSettingsCallback(settingsNamesPerDay, "5")),
			tgbotapi.NewInlineKeyboardButtonData("🔟 (10 дней)", buildSettingsCallback(settingsNamesPerDay, "10")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад к настройкам", buildSettingsCallback(settingsMenu)),
		),
	)
}

// buildQuizModeKeyboard builds keyboard for quiz mode setting.
func buildQuizModeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕 Только новые", buildSettingsCallback(settingsQuizMode, "new")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Только повторение", buildSettingsCallback(settingsQuizMode, "review")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Смешанный режим", buildSettingsCallback(settingsQuizMode, "mixed")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Ежедневный", buildSettingsCallback(settingsQuizMode, "daily")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад к настройкам", buildSettingsCallback(settingsMenu)),
		),
	)
}
