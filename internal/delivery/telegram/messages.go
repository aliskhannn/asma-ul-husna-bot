package telegram

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres/repository"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

// Input / validation.
const (
	msgIncorrectNameNumber  = "Некорректный ввод. Введите число от 1 до 99."
	msgOutOfRangeNumber     = "Номер имени должен быть от 1 до 99."
	msgInvalidRange         = "Некорректный диапазон. Пример: 25 30."
	msgInvalidIntervalHours = "Неверный интервал часов. Выберите 1, 2, 3 или 4."
)

// Data / service errors.
const (
	msgNameUnavailable     = "Не удалось получить имя. Попробуйте позже."
	msgProgressUnavailable = "Не удалось получить прогресс. Попробуйте позже."
	msgSettingsUnavailable = "Не удалось получить настройки. Попробуйте позже."
	msgQuizUnavailable     = "Не удалось создать квиз, попробуйте позже."
	msgInternalError       = "Что‑то пошло не так. Попробуйте позже."
)

// Command/help text.
const (
	msgUnknownCommand = "Неизвестная команда. Список доступных команд:\n\n" +
		"/start — начать работу с ботом\n" +
		"/today — имена на сегодня\n" +
		"/random — случайное имя (guided: из сегодняшних, free: из всех 99)\n" +
		"/quiz — пройти квиз по изучаемым именам\n" +
		"/all — посмотреть все 99 имён\n" +
		"/progress — показать статистику прогресса\n" +
		"/settings — настройки (режим обучения, квиз, напоминания, имён в день)\n" +
		"/help — помощь и список команд\n" +
		"/reset — сбросить прогресс и настройки\n\n" +
		"💡 Также можно:\n" +
		"• Отправить число 1–99, чтобы открыть конкретное имя.\n" +
		"• Отправить диапазон «N M» (например, 5 10), чтобы открыть имена с N по M."
)

const (
	lrm          = "\u200E"
	namesPerPage = 3
)

// md escapes plain text for MarkdownV2.
func md(s string) string {
	return tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, s)
}

func bold(s string) string {
	return "*" + md(s) + "*"
}

// newMessage creates a message with MarkdownV2 parse mode.
func newMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	return msg
}

// newPlainMessage creates a plain message without MarkdownV2 parse mode.
func newPlainMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	return msg
}

// newEdit creates an edit with MarkdownV2 parse mode.
func newEdit(chatID int64, msgID int, text string) tgbotapi.EditMessageTextConfig {
	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = tgbotapi.ModeMarkdownV2
	return edit
}

func msgNoAvailableQuestions() string {
	var sb strings.Builder

	sb.WriteString(md("Пока нет доступных вопросов для квиза."))
	sb.WriteString("\n\n")

	sb.WriteString(md("💡 В управляемом режиме (Guided) квиз использует имена из вашего ежедневного плана. План формируется автоматически по настройке «имён в день»."))
	sb.WriteString("\n\n")

	sb.WriteString(md("Что можно сделать:"))
	sb.WriteString("\n")
	sb.WriteString(md("• Откройте /today и изучайте имена на сегодня\n"))
	sb.WriteString(md("• Переключитесь на свободный режим (Free) в /settings\n"))
	sb.WriteString(md("• Увеличьте «имён в день» в /settings"))

	return sb.String()
}

func msgNoReviews() string {
	var sb strings.Builder

	sb.WriteString(md("Повторений на сегодня нет — все имена свежи в памяти! 🌟"))
	sb.WriteString("\n\n")
	sb.WriteString(md("Попробуйте:"))
	sb.WriteString("\n")
	sb.WriteString(md("• Режим «Смешанный» для практики\n"))
	sb.WriteString(md("• Зайдите позже, когда подойдет время повторения"))

	return sb.String()
}

func msgNoNewNames() string {
	var sb strings.Builder

	sb.WriteString(md("Вы изучили все 99 имён! Ма ша Аллах! 🎉"))
	sb.WriteString("\n\n")
	sb.WriteString(md("Продолжайте повторять в режиме «Повторение» или «Смешанный»."))

	return sb.String()
}

// welcomeMessage builds welcome message safely for MarkdownV2.
// welcomeMessage builds welcome message safely for MarkdownV2.
func welcomeMessage(isNewUser bool, stats *service.ProgressSummary) string {
	var sb strings.Builder

	sb.WriteString(md("السلام عليكم ورحمة الله وبركاته"))
	sb.WriteString("\n\n")

	// returning user
	if !isNewUser && stats != nil {
		sb.WriteString(bold("С возвращением!"))
		sb.WriteString("\n\n")
		sb.WriteString(md(fmt.Sprintf("📊 Ваш прогресс: %d/99 имён выучено (%.1f%%)",
			stats.Learned, stats.Percentage)))
		sb.WriteString("\n\n")

		if stats.DueToday > 0 {
			sb.WriteString(md(fmt.Sprintf("🔄 Сегодня на повторение: %d %s",
				stats.DueToday, formatNamesCount(stats.DueToday))))
			sb.WriteString("\n\n")
			sb.WriteString(bold("Продолжайте с кнопок ниже"))
		} else {
			sb.WriteString(bold("Начните с кнопок ниже"))
		}

		return sb.String()
	}

	return onboardingStep1Message()
}

func helpMessage() string {
	var sb strings.Builder

	sb.WriteString("🤲 ")
	sb.WriteString(bold("Как пользоваться ботом"))
	sb.WriteString("\n\n")

	sb.WriteString("⚡ ")
	sb.WriteString(bold("Быстрый старт:"))
	sb.WriteString("\n")
	sb.WriteString(bold("/today → /quiz → /progress"))
	sb.WriteString(md(" — базовый ежедневный цикл."))
	sb.WriteString("\n\n")

	sb.WriteString("📚 ")
	sb.WriteString(bold("Изучение:"))
	sb.WriteString("\n")
	sb.WriteString("/today — ")
	sb.WriteString(md("имена на сегодня (план формируется автоматически по «имён в день»)"))
	sb.WriteString("\n")
	sb.WriteString("/quiz — ")
	sb.WriteString(md("проверить знания"))
	sb.WriteString("\n\n")

	sb.WriteString("👀 ")
	sb.WriteString(bold("Просто посмотреть (без влияния на прогресс):"))
	sb.WriteString("\n")
	sb.WriteString("/all — ")
	sb.WriteString(md("листать все 99 имён"))
	sb.WriteString("\n")
	sb.WriteString("/random — ")
	sb.WriteString(md("случайное имя"))
	sb.WriteString("\n")
	sb.WriteString("1\\-99 — ")
	sb.WriteString(md("конкретное имя по номеру"))
	sb.WriteString("\n")
	sb.WriteString("N M — ")
	sb.WriteString(md("показать имена в диапазоне (N и M в пределах 1-99)"))
	sb.WriteString("\n")
	sb.WriteString(md("Пример: "))
	sb.WriteString(bold("5 10"))
	sb.WriteString(md(" — имена с 5 по 10"))
	sb.WriteString("\n\n")

	sb.WriteString("⚙️ ")
	sb.WriteString(bold("Прогресс и настройки:"))
	sb.WriteString("\n")
	sb.WriteString("/progress — ")
	sb.WriteString(md("статистика"))
	sb.WriteString("\n")
	sb.WriteString("/settings — ")
	sb.WriteString(md("режим, квиз, напоминания, имён в день"))
	sb.WriteString("\n\n")

	sb.WriteString(md("❓ Остались вопросы? Напишите @husna_support"))

	return sb.String()
}

func learningModeDescription() string {
	var sb strings.Builder

	sb.WriteString("🎯 ")
	sb.WriteString(bold("Управляемый режим"))
	sb.WriteString(" ")
	sb.WriteString(md("(Guided)"))
	sb.WriteString(":\n")
	sb.WriteString(md("• Имена добавляются автоматически по настройке «имён в день»\n"))
	sb.WriteString(md("• /today — основной экран: листайте имена на сегодня и слушайте аудио\n"))
	sb.WriteString(md("• Квиз помогает закреплять изученное и повторять по расписанию (SRS)\n"))
	sb.WriteString(md("• Рекомендуется для постепенного системного изучения"))
	sb.WriteString("\n\n")

	sb.WriteString("🆓 ")
	sb.WriteString(bold("Свободный режим"))
	sb.WriteString(" ")
	sb.WriteString(md("(Free)"))
	sb.WriteString(":\n")
	sb.WriteString(md("• Можно учить в более свободном темпе\n"))
	sb.WriteString(md("• /random и просмотр 1–99 не влияют на прогресс\n"))
	sb.WriteString(md("• Напоминания и настройки работают как обычно\n\n"))

	sb.WriteString("💡 ")
	sb.WriteString(bold("Команды просмотра\n"))
	sb.WriteString(md(" (/random, 1-99, /all) "))
	sb.WriteString(bold("не влияют "))
	sb.WriteString(md("на прогресс в обоих режимах"))

	return sb.String()
}

func formatLearningMode(mode entities.LearningMode) string {
	switch mode {
	case entities.ModeGuided:
		return "🎯 Управляемый"
	case entities.ModeFree:
		return "🆓 Свободный"
	default:
		return string(mode)
	}
}

// formatNameMessage formats a single name message (MarkdownV2 safe).
func formatNameMessage(name *entities.Name) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(
		"%s%s%s %s\n\n%s %s\n%s %s\n\n%s %s",
		lrm,
		md(fmt.Sprintf("%d", name.Number)),
		md("."),
		bold(name.ArabicName),
		md("Транслитерация:"),
		bold(name.Transliteration),
		md("Перевод:"),
		bold(name.Translation),
		md("Значение:"),
		bold(name.Meaning),
	))

	return sb.String()
}

// buildNameResponse builds name message and optional audio.
func buildNameResponse(
	ctx context.Context,
	get func(ctx2 context.Context) (*entities.Name, error),
	chatID int64,
) (tgbotapi.MessageConfig, *tgbotapi.AudioConfig, error) {
	name, err := get(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidNumber) {
			msg := newPlainMessage(chatID, msgIncorrectNameNumber)
			return msg, nil, nil
		}

		if errors.Is(err, repository.ErrNameNotFound) {
			msg := newPlainMessage(chatID, msgNameUnavailable)
			return msg, nil, nil
		}

		msg := newPlainMessage(chatID, msgNameUnavailable)
		return msg, nil, err
	}

	msg := newMessage(chatID, formatNameMessage(name))

	if name.Audio == "" {
		return msg, nil, nil
	}

	audio := buildNameAudio(name, chatID)
	return msg, audio, nil
}

// buildNameAudio creates audio config for a name.
func buildNameAudio(name *entities.Name, chatID int64) *tgbotapi.AudioConfig {
	path := filepath.Join("assets", "audio", name.Audio)
	a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(path))
	a.Caption = name.Transliteration
	return &a
}

// buildNamesPage builds a page of names.
func buildNamesPage(names []*entities.Name, page int) (text string, totalPages int) {
	totalPages = (len(names) + namesPerPage - 1) / namesPerPage
	if totalPages == 0 {
		return "", 0
	}

	pageNames := paginateNames(names, page, namesPerPage)
	var b strings.Builder
	for i, name := range pageNames {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(formatNameMessage(name))
	}

	return b.String(), totalPages
}

func buildNameCardText(name *entities.Name) string {
	return formatNameMessage(name)
}

// buildRangePages builds pages for a range of names.
func buildRangePages(names []*entities.Name, from, to int) (pages []string) {
	if from < 1 {
		from = 1
	}
	if to > len(names) {
		to = len(names)
	}
	if from > to {
		return nil
	}

	fromIdx := from - 1
	toIdx := to

	for start := fromIdx; start < toIdx; start += namesPerPage {
		end := start + namesPerPage
		if end > toIdx {
			end = toIdx
		}

		chunk := names[start:end]
		var b strings.Builder
		for i, name := range chunk {
			if i > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(formatNameMessage(name))
		}

		pages = append(pages, b.String())
	}

	return pages
}

// paginateNames returns a slice of names for a given page.
func paginateNames(names []*entities.Name, page, namesPerPage int) []*entities.Name {
	start := page * namesPerPage
	end := start + namesPerPage

	if start >= len(names) {
		return nil
	}

	if end > len(names) {
		end = len(names)
	}

	return names[start:end]
}

// getAllNames retrieves all names from the service.
func (h *Handler) getAllNames(ctx context.Context) ([]*entities.Name, error) {
	names, err := h.nameService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	if len(names) == 0 {
		return nil, nil
	}

	return names, nil
}

// buildProgressBar creates an ASCII progress bar.
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

// buildQuizStartMessage builds quiz start message (MarkdownV2 safe).
func buildQuizStartMessage(mode string) string {
	modeText := formatQuizMode(mode)

	return fmt.Sprintf(
		"%s\n\n%s %s\n\n%s",
		bold("🎯 Квиз начинается!"),
		md("Режим:"),
		bold(modeText),
		md("Выберите правильный вариант ответа для каждого вопроса."),
	)
}

// formatQuizMode formats quiz mode for display.
func formatQuizMode(mode string) string {
	switch mode {
	case "new":
		return "🆕 Только новые"
	case "review":
		return "🔄 Только повторение"
	case "mixed":
		return "🎲 Смешанный"
	default:
		return mode
	}
}

// formatQuizResult formats quiz results (MarkdownV2 safe).
func formatQuizResult(session *entities.QuizSession) string {
	percentage := float64(session.CorrectAnswers) / float64(session.TotalQuestions) * 100

	emoji, message := "📚", "Продолжайте изучать имена Аллаха!"
	switch {
	case percentage >= 90:
		emoji, message = "🌟", "Отличный результат! Ма ша Аллах!"
	case percentage >= 70:
		emoji, message = "👍", "Хороший результат!"
	case percentage >= 50:
		emoji, message = "💪", "Неплохо, продолжайте!"
	}

	progressBar := buildProgressBar(session.CorrectAnswers, session.TotalQuestions, 10)

	return fmt.Sprintf(
		"%s %s\n\n%s %s\n%s\n\n%s",
		md(emoji),
		md("Квиз завершён!"),
		md("Результат:"),
		bold(fmt.Sprintf("%d/%d (%.0f%%)", session.CorrectAnswers, session.TotalQuestions, percentage)),
		md(progressBar),
		md(message),
	)
}

// formatAnswerFeedback formats feedback for a quiz answer (MarkdownV2 safe).
func formatAnswerFeedback(isCorrect bool, correctAnswer string) string {
	if isCorrect {
		return md("✅ Правильно!")
	}
	return fmt.Sprintf(
		"%s\n\n%s %s",
		md("❌ Неправильно"),
		md("Правильный ответ:"),
		bold(correctAnswer),
	)
}

// formatProgressMessage formats the progress summary for display.
func formatProgressMessage(summary *service.ProgressSummary, progressBar string) string {
	var sb strings.Builder

	sb.WriteString("📊 ")
	sb.WriteString(bold("Ваш прогресс"))
	sb.WriteString("\n\n")

	sb.WriteString(md(progressBar))
	sb.WriteString("\n\n")

	sb.WriteString(md(fmt.Sprintf("✅ Выучено: %d/99 (%.1f%%)\n",
		summary.Learned, summary.Percentage)))

	sb.WriteString(md(fmt.Sprintf("📚 В процессе: %d/99\n", summary.InProgress)))

	if summary.InProgress > 0 {
		sb.WriteString(md(fmt.Sprintf("  ├─ 🆕 Новые: %d\n", summary.NewCount)))
		sb.WriteString(md(fmt.Sprintf("  └─ 📖 Изучаются: %d\n", summary.LearningCount)))
	}

	sb.WriteString(md(fmt.Sprintf("⭕ Не начато: %d/99\n", summary.NotStarted)))

	sb.WriteString("\n")

	if summary.DueToday > 0 {
		sb.WriteString(md(fmt.Sprintf("🔄 Повторений сегодня: %d\n", summary.DueToday)))
	}

	if summary.Learned > 0 {
		sb.WriteString(md(fmt.Sprintf("🎯 Точность: %.1f%%\n", summary.Accuracy)))
	}

	if summary.DaysToComplete > 0 {
		sb.WriteString(md(fmt.Sprintf("📅 Примерно дней до финиша: %d", summary.DaysToComplete)))
	}

	return sb.String()
}

// buildReminderSettingsMessage builds reminder settings screen message
func buildReminderSettingsMessage(timezone string, reminder *entities.UserReminders) string {
	if reminder == nil {
		return md("⏰ Настройки напоминаний") + "\n\n" +
			md("Статус: ") + bold("🔕 Отключены") + "\n\n" +
			md("Напоминания помогут не забывать о ежедневной практике изучения имён Аллаха.")
	}

	status := "🔕 Отключены"
	details := ""

	if reminder.IsEnabled {
		status = "🔔 Включены"

		freqText := formatIntervalHoursInt(reminder.IntervalHours)

		startTime := reminder.StartTime[:5] // "08:00"
		endTime := reminder.EndTime[:5]     // "20:00"

		details = fmt.Sprintf(
			"\n%s %s\n%s %s\n%s %s — %s",
			md("🌍 Часовой пояс:"),
			bold(timezone),
			md("📅 Частота:"),
			bold(freqText),
			md("⏰ Время:"),
			bold(startTime),
			bold(endTime),
		)
	}

	return fmt.Sprintf(
		"%s\n\n%s %s%s\n\n%s",
		md("⏰ Настройки напоминаний"),
		md("Статус:"),
		bold(status),
		details,
		md("Напоминания помогут не забывать о ежедневной практике изучения имён Аллаха."),
	)
}

func buildTimezoneMenuMessage(current string) string {
	if current == "" {
		current = "UTC"
	}

	var sb strings.Builder
	sb.WriteString(md("🌍 "))
	sb.WriteString(bold("Часовой пояс"))
	sb.WriteString("\n\n")
	sb.WriteString(md("Текущий: "))
	sb.WriteString(bold(current))
	sb.WriteString("\n\n")
	sb.WriteString(md("Выберите смещение от UTC, чтобы напоминания приходили по местному времени."))

	return sb.String()
}

// formatIntervalHoursInt formats interval hours for display.
func formatIntervalHoursInt(freq int) string {
	switch freq {
	case 1:
		return "Каждый час"
	case 2:
		return "Каждые 2 часа"
	case 3:
		return "Каждые 3 часа"
	case 4:
		return "Каждые 4 часа"
	default:
		return fmt.Sprintf("Каждые %d часа", freq)
	}
}

// formatIntervalHoursString converts interval string to integer.
func formatIntervalHoursString(freq string) (int, error) {
	switch freq {
	case "every_1h":
		return 1, nil
	case "every_2h":
		return 2, nil
	case "every_3h":
		return 3, nil
	case "every_4h":
		return 4, nil
	default:
		return 0, fmt.Errorf("invalid frequency %q", freq)
	}
}

// formatReminderStatus formats reminder status for settings display
func formatReminderStatus(reminder *entities.UserReminders) string {
	if reminder == nil || !reminder.IsEnabled {
		return "🔕 Отключены"
	}

	freqText := formatIntervalHoursInt(reminder.IntervalHours)

	startTime := reminder.StartTime[:5] // "08:00"
	endTime := reminder.EndTime[:5]     // "20:00"

	return fmt.Sprintf("🔔 %s в день (%s-%s)", freqText, startTime, endTime)
}

// buildReminderNotification builds reminder notification message.
func buildReminderNotification(payload entities.ReminderPayload) string {
	var sb strings.Builder

	switch payload.Kind {
	case entities.ReminderKindReview:
		sb.WriteString(md("🔔 "))
		sb.WriteString(bold("Время повторить имена Аллаха!"))
		sb.WriteString("\n\n")
		sb.WriteString(md("📖 Имя для повторения:"))
	case entities.ReminderKindStudy:
		sb.WriteString(md("📚 "))
		sb.WriteString(bold("Время продолжить изучение сегодняшних имён!"))
		sb.WriteString("\n\n")
		sb.WriteString(md("📖 Имя на сегодня:"))
	case entities.ReminderKindNew:
		fallthrough
	default:
		sb.WriteString(md("🌟 "))
		sb.WriteString(bold("Время узнать новое имя Аллаха!"))
		sb.WriteString("\n\n")
		sb.WriteString(md("📖 Имя на сегодня:"))
	}

	sb.WriteString("\n\n")

	sb.WriteString(formatNameMessage(&payload.Name))
	sb.WriteString("\n\n")

	sb.WriteString(md("📊 "))
	sb.WriteString(bold("Ваш прогресс:"))
	sb.WriteString("\n\n")

	if payload.Stats.DueToday > 0 {
		sb.WriteString(md(fmt.Sprintf("🔄 Повторов сегодня: %d\n", payload.Stats.DueToday)))
	}

	sb.WriteString(md(fmt.Sprintf("✅ Выучено: %d/99\n", payload.Stats.Learned)))

	if payload.Stats.NotStarted > 0 {
		sb.WriteString(md(fmt.Sprintf("🆕 Не начато: %d\n", payload.Stats.NotStarted)))
	}

	if payload.Stats.DaysToComplete > 0 {
		sb.WriteString(md(fmt.Sprintf("📅 Примерно дней до финиша: %d", payload.Stats.DaysToComplete)))
	}

	return sb.String()
}

func buildFirstQuizMessage() string {
	var sb strings.Builder

	sb.WriteString(md("💡 "))
	sb.WriteString(bold("Как работает квиз:"))
	sb.WriteString("\n")
	sb.WriteString(md("• Выберите правильный ответ из вариантов\n"))
	sb.WriteString(md("• 2+ правильных ответа = имя начнёт изучаться\n"))
	sb.WriteString(md("• Я буду повторять имена по графику"))

	return sb.String()
}

// buildQuizQuestionText formats quiz question text from database question.
func buildQuizQuestionText(
	question *entities.QuizQuestion,
	name *entities.Name,
	currentNum, totalQuestions int,
) string {
	var sb strings.Builder

	sb.WriteString(md(fmt.Sprintf("Вопрос %d из %d", currentNum, totalQuestions)))
	sb.WriteString("\n\n")

	var questionPrompt string
	switch question.QuestionType {
	case string(entities.QuestionTypeTranslation):
		questionPrompt = fmt.Sprintf("Какое арабское имя означает: %s?", name.Translation)
	case string(entities.QuestionTypeTransliteration):
		questionPrompt = fmt.Sprintf("Что означает имя %s?", name.Transliteration)
	case string(entities.QuestionTypeMeaning):
		questionPrompt = fmt.Sprintf("Какое из имён соответствует значению: %s?", name.Meaning)
	case string(entities.QuestionTypeArabic):
		questionPrompt = fmt.Sprintf("Что означает арабское имя %s?", name.ArabicName)
	default:
		questionPrompt = name.ArabicName
	}

	sb.WriteString(bold(questionPrompt))

	return sb.String()
}

func formatNamesCount(n int) string {
	if n == 1 {
		return "имя"
	}
	if n >= 2 && n <= 4 {
		return "имени"
	}
	return "имён"
}
