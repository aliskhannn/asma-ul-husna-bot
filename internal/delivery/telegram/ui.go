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
			tgbotapi.NewInlineKeyboardButtonData("📝 Длина квиза", buildSettingsCallback(settingsQuizLength)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Режим квиза", buildSettingsCallback(settingsQuizMode)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔤 Транслитерация", buildSettingsCallback(settingsToggleTransliteration)),
			tgbotapi.NewInlineKeyboardButtonData("🔊 Аудио", buildSettingsCallback(settingsToggleAudio)),
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

// buildQuizLengthKeyboard builds keyboard for quiz length setting.
func buildQuizLengthKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("5 вопросов", buildSettingsCallback(settingsQuizLength, "5")),
			tgbotapi.NewInlineKeyboardButtonData("10 вопросов", buildSettingsCallback(settingsQuizLength, "10")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("15 вопросов", buildSettingsCallback(settingsQuizLength, "15")),
			tgbotapi.NewInlineKeyboardButtonData("20 вопросов", buildSettingsCallback(settingsQuizLength, "20")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("25 вопросов", buildSettingsCallback(settingsQuizLength, "25")),
			tgbotapi.NewInlineKeyboardButtonData("30 вопросов", buildSettingsCallback(settingsQuizLength, "30")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("35 вопросов", buildSettingsCallback(settingsQuizLength, "35")),
			tgbotapi.NewInlineKeyboardButtonData("40 вопросов", buildSettingsCallback(settingsQuizLength, "40")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("45 вопросов", buildSettingsCallback(settingsQuizLength, "45")),
			tgbotapi.NewInlineKeyboardButtonData("50 вопросов", buildSettingsCallback(settingsQuizLength, "50")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад к настройкам", buildSettingsCallback(settingsMenu)),
		),
	)
}

// buildToggleTransliterationKeyboard builds keyboard for transliteration toggle.
func buildToggleTransliterationKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Показывать транслитерацию", buildSettingsCallback(settingsToggleTransliteration, "on")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Не показывать", buildSettingsCallback(settingsToggleTransliteration, "off")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад к настройкам", buildSettingsCallback(settingsMenu)),
		),
	)
}

// buildToggleAudioKeyboard builds keyboard for audio toggle.
func buildToggleAudioKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔊 Включить аудио", buildSettingsCallback(settingsToggleAudio, "on")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔇 Отключить аудио", buildSettingsCallback(settingsToggleAudio, "off")),
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
			tgbotapi.NewInlineKeyboardButtonData("🆕 Только новые", buildSettingsCallback(settingsQuizMode, "new_only")),
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
