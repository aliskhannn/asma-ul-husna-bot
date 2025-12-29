package telegram

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type OnboardingState struct {
	Step        int
	NamesPerDay int
}

type OnboardingStep int

const (
	StepWelcome OnboardingStep = iota
	StepNamesPerDay
	StepLearningMode
	StepReminders
	StepTimezone
	StepComplete
)

func (s OnboardingStep) Message() string {
	switch s {
	case StepWelcome:
		return onboardingStep1Message()
	case StepNamesPerDay:
		return onboardingStep2Message()
	case StepLearningMode:
		return onboardingStep3Message()
	case StepReminders:
		return onboardingStep4Message()
	case StepTimezone:
		return onboardingStepTimezoneMessage()
	case StepComplete:
		return onboardingCompleteMessage()
	}

	return ""
}

func onboardingStep1Message() string {
	var sb strings.Builder

	sb.WriteString(md("السلام عليكم ورحمة الله وبركاته"))
	sb.WriteString("\n\n")
	sb.WriteString(bold("Добро пожаловать в Asma ul Husna Bot!"))
	sb.WriteString("\n\n")
	sb.WriteString(md("Я помогу вам выучить 99 прекрасных имён Аллаха через:"))
	sb.WriteString("\n")
	sb.WriteString(md("📖 Карточки с переводом и аудио\n"))
	sb.WriteString(md("🧠 Умные квизы\n"))
	sb.WriteString(md("⏰ Напоминания для повторения\n"))
	sb.WriteString("\n")
	sb.WriteString(md("Сейчас настроим бота под вас за 3 простых шага ⬇️"))

	return sb.String()
}

func onboardingStep1Keyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Начать настройку 🚀", buildOnboardingStepCallback(2)),
		),
	)
}

func onboardingStep2Message() string {
	var sb strings.Builder

	sb.WriteString(md("Шаг 1 из 3"))
	sb.WriteString("\n\n")
	sb.WriteString(bold("Сколько новых имён в день вы готовы изучать?"))
	sb.WriteString("\n\n")
	sb.WriteString(md("💡 Рекомендуем начать с 1-2 имён — это оптимально для запоминания"))

	return sb.String()
}

func onboardingStep2Keyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 имя/день (99 дней)", buildOnboardingNamesPerDayCallback(1)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("2 имени/день ⭐ (50 дней)", buildOnboardingNamesPerDayCallback(2)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3 имени/день (33 дня)", buildOnboardingNamesPerDayCallback(3)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("5 имён/день (20 дней)", buildOnboardingNamesPerDayCallback(5)),
		),
	)
}

func onboardingStep3Message() string {
	var sb strings.Builder

	sb.WriteString(md("Шаг 2 из 3"))
	sb.WriteString("\n\n")
	sb.WriteString(bold("Выберите режим обучения:"))
	sb.WriteString("\n\n")

	// Guided
	sb.WriteString("🎯 ")
	sb.WriteString(bold("Управляемый"))
	sb.WriteString(md(" (рекомендуется)\n"))
	sb.WriteString(md("• Имена добавляются автоматически по настройке «имён в день»\n"))
	sb.WriteString(md("• Основной экран — /today: листайте имена на сегодня и слушайте аудио\n"))
	sb.WriteString(md("• Квиз помогает закреплять изученное и повторять по расписанию (SRS)\n"))
	sb.WriteString("\n")

	// Free
	sb.WriteString("🆓 ")
	sb.WriteString(bold("Свободный\n"))
	sb.WriteString(md("• Изучайте в своём темпе\n"))
	sb.WriteString(md("• Можно чаще пользоваться /random и просмотром 1–99 (это не влияет на прогресс)\n"))
	sb.WriteString(md("• Для тех, кто хочет больше гибкости"))

	return sb.String()
}

func onboardingStep3Keyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎯 Управляемый ⭐", buildOnboardingModeCallback("guided")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆓 Свободный", buildOnboardingModeCallback("free")),
		),
	)
}

func onboardingStep4Message() string {
	var sb strings.Builder

	sb.WriteString(md("Шаг 3 из 3"))
	sb.WriteString("\n\n")
	sb.WriteString(bold("Включить напоминания?"))
	sb.WriteString("\n\n")
	sb.WriteString(md("⏰ Я буду напоминать вам про имена в удобное время"))
	sb.WriteString("\n\n")
	sb.WriteString(md("💡 Можно настроить позже в /settings"))

	return sb.String()
}

func onboardingStep4Keyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔔 Да, включить", buildOnboardingRemindersCallback("yes")),
			tgbotapi.NewInlineKeyboardButtonData("Пока нет", buildOnboardingRemindersCallback("no")),
		),
	)
}

func onboardingStepTimezoneMessage() string {
	var sb strings.Builder
	sb.WriteString(md("Шаг 3 из 3"))
	sb.WriteString("\n\n")
	sb.WriteString(bold("Выберите ваш часовой пояс (UTC)"))
	sb.WriteString("\n\n")
	sb.WriteString(md("Это нужно, чтобы напоминания приходили по местному времени.\n"))
	sb.WriteString(md("Пример: Москва — UTC+3."))
	sb.WriteString("\n\n")
	sb.WriteString(md("Можно поменять позже в /settings."))
	return sb.String()
}

func onboardingStepTimezoneKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+0", buildOnboardingTimezoneCallback("UTC+0")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+1", buildOnboardingTimezoneCallback("UTC+1")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+2", buildOnboardingTimezoneCallback("UTC+2")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+3", buildOnboardingTimezoneCallback("UTC+3")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+4", buildOnboardingTimezoneCallback("UTC+4")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+5", buildOnboardingTimezoneCallback("UTC+5")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+6", buildOnboardingTimezoneCallback("UTC+6")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+7", buildOnboardingTimezoneCallback("UTC+7")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+8", buildOnboardingTimezoneCallback("UTC+8")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("UTC+9", buildOnboardingTimezoneCallback("UTC+9")),
			tgbotapi.NewInlineKeyboardButtonData("UTC+10", buildOnboardingTimezoneCallback("UTC+10")),
			tgbotapi.NewInlineKeyboardButtonData("Другой", buildOnboardingTimezoneCallback("manual")),
		),
	)
}

func onboardingCompleteMessage() string {
	var sb strings.Builder

	sb.WriteString(md("✅ "))
	sb.WriteString(bold("Всё готово!"))
	sb.WriteString("\n\n")

	sb.WriteString(md("Ваш ежедневный план формируется автоматически по настройке «имён в день»."))
	sb.WriteString("\n\n")

	sb.WriteString(md("Начните прямо сейчас:"))
	sb.WriteString("\n\n")

	sb.WriteString(bold("1️⃣ /today"))
	sb.WriteString(md(" — ваши имена на сегодня\n"))
	sb.WriteString(bold("2️⃣ /quiz"))
	sb.WriteString(md(" — проверит, как вы запомнили\n"))
	sb.WriteString(bold("3️⃣ /progress"))
	sb.WriteString(md(" — покажет статистику\n"))
	sb.WriteString("\n")

	sb.WriteString(md("📖 "))
	sb.WriteString(bold("Хотите просто посмотреть все имена?"))
	sb.WriteString("\n")
	sb.WriteString(md("Используйте /all — это не повлияет на обучение!"))
	sb.WriteString("\n\n")

	sb.WriteString(md("💡 Совет: откройте /today и пройдитесь по именам дня."))
	return sb.String()
}

func onboardingCompleteKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Открыть /today", buildOnboardingCmdCallback("today")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👀 Посмотреть все имена", buildOnboardingCmdCallback("all")),
		),
	)
}
