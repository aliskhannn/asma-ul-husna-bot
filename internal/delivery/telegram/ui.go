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
			tgbotapi.NewInlineKeyboardButtonData("🎯 Режим обучения", buildSettingsCallback(settingsLearningMode)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Режим квиза", buildSettingsCallback(settingsQuizMode)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Напоминания", buildSettingsCallback(settingsReminders)),
		),
	)
}

func buildLearningModeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🎯 Управляемый (рекомендуется)",
				buildSettingsCallback(settingsLearningMode, "guided"),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🆓 Свободный",
				buildSettingsCallback(settingsLearningMode, "free"),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад к настройкам", buildSettingsCallback(settingsMenu)),
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
func buildQuizAnswerKeyboard(sessionID int64, questionNum int, options []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, option := range options {
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
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3️⃣ (33 дня)", buildSettingsCallback(settingsNamesPerDay, "3")),
			tgbotapi.NewInlineKeyboardButtonData("5️⃣ (20 дней)", buildSettingsCallback(settingsNamesPerDay, "5")),
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
			tgbotapi.NewInlineKeyboardButtonData("🎲 Смешанный", buildSettingsCallback(settingsQuizMode, "mixed")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад к настройкам", buildSettingsCallback(settingsMenu)),
		),
	)
}

// buildRemindersKeyboard builds the reminder settings keyboard.
func buildRemindersKeyboard(reminder *entities.UserReminders) tgbotapi.InlineKeyboardMarkup {
	enabled := reminder != nil && reminder.IsEnabled

	toggleText := "🔕 Отключить"
	if !enabled {
		toggleText = "🔔 Включить"
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(toggleText, buildReminderToggleCallback()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Часовой пояс", buildSettingsCallback(settingsReminders, "timezone")),
		),
	}

	if enabled {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📅 Частота", buildSettingsCallback(settingsReminders, "frequency")),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏰ Время", buildSettingsCallback(settingsReminders, "time")),
			),
		)
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад к настройкам", buildSettingsCallback(settingsMenu)),
		),
	)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// buildTimezoneKeyboard builds a simple UTC offset picker for MVP.
func buildTimezoneKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+0", buildSettingsCallback(settingsReminders, "tz", "UTC+0")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+1", buildSettingsCallback(settingsReminders, "tz", "UTC+1")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+2", buildSettingsCallback(settingsReminders, "tz", "UTC+2")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+3", buildSettingsCallback(settingsReminders, "tz", "UTC+3")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+4", buildSettingsCallback(settingsReminders, "tz", "UTC+4")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+5", buildSettingsCallback(settingsReminders, "tz", "UTC+5")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+6", buildSettingsCallback(settingsReminders, "tz", "UTC+6")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+7", buildSettingsCallback(settingsReminders, "tz", "UTC+7")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+8", buildSettingsCallback(settingsReminders, "tz", "UTC+8")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+9", buildSettingsCallback(settingsReminders, "tz", "UTC+9")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+10", buildSettingsCallback(settingsReminders, "tz", "UTC+10")),
			tgbotapi.NewInlineKeyboardButtonData("Другой", buildSettingsCallback(settingsReminders, "timezone_manual")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад", buildSettingsCallback(settingsReminders)),
		),
	)
}

// buildReminderKeyboard builds keyboard for reminder notification
func buildReminderKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Начать квиз", buildReminderStartQuizCallback()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Напомнить позже", buildReminderSnoozeCallback()),
			tgbotapi.NewInlineKeyboardButtonData("🔕 Отключить", buildReminderDisableCallback()),
		),
	)
}

// buildFrequencyKeyboard builds keyboard for frequency selection
func buildFrequencyKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Каждый час", buildSettingsCallback(settingsReminders, "freq", "every_1h")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Каждые 2 часа", buildSettingsCallback(settingsReminders, "freq", "every_2h")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Каждые 3 часа", buildSettingsCallback(settingsReminders, "freq", "every_3h")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Каждые 4 часа", buildSettingsCallback(settingsReminders, "freq", "every_4h")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад", buildSettingsCallback(settingsReminders)),
		),
	)
}

// buildTimeWindowKeyboard builds keyboard for time window selection
func buildTimeWindowKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌅 Утро (08:00-12:00)", buildSettingsCallback(settingsReminders, "time", "08-00-00", "12-00-00")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("☀️ День (12:00-18:00)", buildSettingsCallback(settingsReminders, "time", "12-00-00", "18-00-00")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌙 Вечер (18:00-22:00)", buildSettingsCallback(settingsReminders, "time", "18-00-00", "22-00-00")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Весь день (08:00-22:00)", buildSettingsCallback(settingsReminders, "time", "08-00-00", "22-00-00")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад", buildSettingsCallback(settingsReminders)),
		),
	)
}

func buildResetKeyboard() *tgbotapi.InlineKeyboardMarkup {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Сбросить", buildResetConfirmCallback()),
			tgbotapi.NewInlineKeyboardButtonData("✅ Отменить", buildResetCancelCallback()),
		),
	)
	return &kb
}

func welcomeReturningKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Открыть /today", buildTodayPageCallback(0)),
			tgbotapi.NewInlineKeyboardButtonData("🎯 Начать квиз", buildQuizStartCallback()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Прогресс", buildProgressCallback()),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", buildSettingsCallback(settingsMenu)),
		),
	)
}

func todayCardsKeyboard(page, total, nameNumber int) *tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	if total > 1 {
		var nav []tgbotapi.InlineKeyboardButton
		if page > 0 {
			nav = append(nav,
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Предыдущее", buildTodayPageCallback(page-1)),
			)
		}
		if page+1 < total {
			nav = append(nav,
				tgbotapi.NewInlineKeyboardButtonData("Следующее ➡️", buildTodayPageCallback(page+1)),
			)
		}
		if len(nav) > 0 {
			rows = append(rows, nav)
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🎯 Начать квиз", buildQuizStartCallback()),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔊 Прослушать", buildTodayAudioCallback(nameNumber)),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", buildSettingsCallback(settingsMenu)),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return &kb
}
