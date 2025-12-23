package telegram

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

// Error messages.
const (
	msgIncorrectNameNumber  = "Некорректный ввод. Введите число от 1 до 99."
	msgOutOfRangeNumber     = "Номер имени должен быть от 1 до 99."
	msgUseRange             = "Используйте: /range 25 30."
	msgInvalidRange         = "Некорректный диапазон. Пример: /range 25 30."
	msgInvalidIntervalHours = "Неверный интервал часов. Выберите 1, 2, 3 или 4."
	msgNameUnavailable      = "Не удалось получить имя. Попробуйте позже."
	msgProgressUnavailable  = "Не удалось получить прогресс. Попробуйте позже."
	msgSettingsUnavailable  = "Не удалось получить настройки. Попробуйте позже."
	msgQuizUnavailable      = "Не удалось создать квиз, попробуйте позже."
	//msgNoAvailableQuestions = "Пока нет доступных вопросов для квиза.\nЗайдите позже или измените режим/количество новых имён в настройках."
	//msgNoReviews      = "Повторений на сегодня нет — все имена пока не требуют повторения.\nПопробуйте режим «Смешанный» или зайдите позже."
	//msgNoNewNames     = "Новых имён больше нет — вы прошли все 99 имён.\nПереключитесь на «Повторение» или «Смешанный», чтобы закреплять."
	msgInternalError  = "Что‑то пошло не так. Попробуйте позже."
	msgUnknownCommand = "Неизвестная команда. Список доступных команд:\n\n" +
		"/start — начать работу с ботом\n" +
		"/next — следующее имя на сегодня (учитывает лимит «имён в день»)\n" +
		"/today — список всех имён на сегодня\n" +
		"/random — случайное имя из сегодняшних (guided) или любое (free)\n" +
		"/quiz — квиз по введённым сегодня именам\n" +
		"/all — посмотреть все 99 имён\n" +
		"/range N M — имена с N по M (например, /range 1 10)\n" +
		"1-99 — просмотр конкретного имени (ознакомление)\n" +
		"/progress — показать статистику прогресса\n" +
		"/settings — настройки (квиз, напоминания, количество имён в день)\n" +
		"/help — помощь и список команд"
)

const (
	lrm          = "\u200E"
	namesPerPage = 5
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
	sb.WriteString(md("💡 В управляемом режиме (Guided) квизы показывают только имена, которые уже были введены через напоминания."))
	sb.WriteString("\n\n")
	sb.WriteString(md("Вы можете:"))
	sb.WriteString("\n")
	sb.WriteString(md("• Дождаться следующего напоминания\n"))
	sb.WriteString(md("• Переключиться на свободный режим (Free) в /settings\n"))
	sb.WriteString(md("• Увеличить количество имен в день"))

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
func welcomeMessage() string {
	var sb strings.Builder

	sb.WriteString(md("السلام عليكم ورحمة الله وبركاته"))
	sb.WriteString("\n\n")

	sb.WriteString(md("Абу Хурайра, да будет доволен им Аллах, передаёт, что Посланник Аллаха ﷺ сказал: «Поистине, у Аллаха девяносто девять имён — сотня без одного, и каждый, кто запомнит их, войдёт в Рай. Поистине, Он (— это Тот, Кто) не имеет пары /витр/, и Он любит (всё) непарное». (Аль-Бухари, 6410)"))
	sb.WriteString("\n\n")

	sb.WriteString(bold("Asma ul Husna Bot"))
	sb.WriteString(md(" поможет вам в изучении "))
	sb.WriteString(bold("99 имён Алла́ха"))
	sb.WriteString(md(" (асма̄'у -лла̄һи ль-х̣усна̄ — «прекраснейшие имена Аллаха»)."))
	sb.WriteString("\n\n")

	sb.WriteString(md("С нами вы сможете:"))
	sb.WriteString("\n\n")

	sb.WriteString(md("📖 Изучать каждое имя с "))
	sb.WriteString(bold("переводом"))
	sb.WriteString(md(", "))
	sb.WriteString(bold("транслитерацией"))
	sb.WriteString(md(" и "))
	sb.WriteString(bold("аудиопроизношением"))
	sb.WriteString(md("."))
	sb.WriteString("\n")

	sb.WriteString(md("⏰ Настроить "))
	sb.WriteString(bold("гибкие напоминания"))
	sb.WriteString(md(" для ежедневного повторения."))
	sb.WriteString("\n")

	sb.WriteString(md("🧠 Проходить "))
	sb.WriteString(bold("квизы"))
	sb.WriteString(md(" для проверки и отслеживания прогресса."))
	sb.WriteString("\n\n")

	sb.WriteString(bold("Чтобы начать:"))
	sb.WriteString("\n\n")

	sb.WriteString(md("📚 "))
	sb.WriteString(bold("Изучение:"))
	sb.WriteString("\n")
	sb.WriteString(md("• /next — "))
	sb.WriteString(bold("следующее имя на сегодня"))
	sb.WriteString(md(" (учитывает лимит «имён в день»)"))
	sb.WriteString("\n")
	sb.WriteString(md("• /today — "))
	sb.WriteString(bold("список всех имён на сегодня"))
	sb.WriteString("\n")
	sb.WriteString(md("• /quiz — "))
	sb.WriteString(bold("квиз по введённым сегодня именам"))
	sb.WriteString("\n")
	sb.WriteString(md("• /random — "))
	sb.WriteString(bold("случайное из сегодняшних"))
	sb.WriteString("\n")
	sb.WriteString(md("• 1-99 — просмотр конкретного имени (ознакомление)"))
	sb.WriteString("\n\n")

	sb.WriteString(md("📊 "))
	sb.WriteString(bold("Прогресс и настройки:"))
	sb.WriteString("\n")
	sb.WriteString(md("• /progress — посмотреть статистику"))
	sb.WriteString("\n")
	sb.WriteString(md("• /settings — настроить квиз, напоминания, "))
	sb.WriteString(bold("имён в день"))
	sb.WriteString("\n")
	sb.WriteString(md("• /help — подробная справка"))
	sb.WriteString("\n\n")

	sb.WriteString(md("💡 "))
	sb.WriteString(bold("Совет:"))
	sb.WriteString(md(" "))
	sb.WriteString(bold("/next → /today → /quiz"))
	sb.WriteString(md(" — идеальный цикл изучения!"))
	sb.WriteString("\n\n")

	sb.WriteString(md("Начните свой путь к знанию прямо сейчас!"))

	return sb.String()
}

func helpMessage() string {
	var sb strings.Builder

	sb.WriteString("🤖 ")
	sb.WriteString(bold("Доступные команды"))
	sb.WriteString("\n\n")

	sb.WriteString("📚 ")
	sb.WriteString(bold("Изучение:"))
	sb.WriteString("/next — ")
	sb.WriteString(bold("следующее имя на сегодня"))
	sb.WriteString(" \\(учитывает лимит «имён в день»\\)\n")
	sb.WriteString("/today — ")
	sb.WriteString(bold("список всех имён на сегодня"))
	sb.WriteString("\n")
	sb.WriteString("/random — ")
	sb.WriteString(bold("случайное из сегодняшних"))
	sb.WriteString(" \\(guided\\) / любое \\(free\\)\n")
	sb.WriteString("/quiz — ")
	sb.WriteString(bold("квиз по введённым сегодня именам"))
	sb.WriteString("\n\n")

	sb.WriteString("📖 ")
	sb.WriteString(bold("Просмотр:"))
	sb.WriteString("/all — Посмотреть все 99 имён\n")
	sb.WriteString("/range N M — имена с N по M \\(например, /range 1 10\\)\n")
	sb.WriteString("1\\-99 — просмотр конкретного имени \\(ознакомление\\)\n\n")

	sb.WriteString("📊 ")
	sb.WriteString(bold("Прогресс:"))
	sb.WriteString("/progress — Посмотреть статистику\n\n")

	sb.WriteString("⚙️ ")
	sb.WriteString(bold("Настройки:"))
	sb.WriteString("/settings — Квиз, напоминания, ")
	sb.WriteString(bold("имён в день"))
	sb.WriteString("\n")
	sb.WriteString("/help — Показать эту справку\n\n")

	sb.WriteString("💡 ")
	sb.WriteString(bold("Как использовать:"))
	sb.WriteString("\n• ")
	sb.WriteString(bold("/next → /today → /quiz"))
	sb.WriteString(" — идеальный цикл изучения\\!\n")
	sb.WriteString("• /random и номера \\(1\\-99\\) ")
	sb.WriteString(bold("не влияют"))
	sb.WriteString(" на прогресс\n")
	sb.WriteString("• Напоминания автоматически вводят новые имена по квоте")

	return sb.String()
}

func learningModeDescription() string {
	var sb strings.Builder

	// Управляемый режим
	sb.WriteString("🎯 ")
	sb.WriteString(bold("Управляемый режим"))
	sb.WriteString(" ")
	sb.WriteString(md("(Guided)"))
	sb.WriteString(":\n")
	sb.WriteString(md("• /next вводит новые имена"))
	sb.WriteString(bold(" по квоте "))
	sb.WriteString(md("«имён в день»\n"))
	sb.WriteString(md("• Квиз показывает"))
	sb.WriteString(bold(" только введённые сегодня "))
	sb.WriteString(md("имена\n"))
	sb.WriteString(md("• После изучения (2+ правильных ответа) → разблокировка следующего\n"))
	sb.WriteString(md("• Строгое следование SRS (интервальное повторение)\n"))
	sb.WriteString(md("• Рекомендуется для глубокого постепенного изучения"))
	sb.WriteString("\n\n")

	// Свободный режим
	sb.WriteString("🆓 ")
	sb.WriteString(bold("Свободный режим"))
	sb.WriteString(" ")
	sb.WriteString(md("(Free)"))
	sb.WriteString(":\n")
	sb.WriteString(md("• Квиз может вводить новые имена"))
	sb.WriteString(bold(" без ограничений\n"))
	sb.WriteString(md("• /next и напоминания работают как обычно\n"))
	sb.WriteString(md("• Гибкий темп, можно учить много за раз\n\n"))

	sb.WriteString("💡 ")
	sb.WriteString(bold("Команды просмотра\n"))
	sb.WriteString(md(" (/random, 1-99) "))
	sb.WriteString(bold("не влияют "))
	sb.WriteString(md("на прогресс в обоих режимах"))

	return sb.String()
}

func formatLearningMode(mode entities.LearningMode) string {
	switch mode {
	case entities.ModeGuided:
		return "🎯 Управляемое (рекомендуется)"
	case entities.ModeFree:
		return "🆓 Свободное"
	default:
		return string(mode)
	}
}

// formatNameMessage formats a single name message (MarkdownV2 safe).
func formatNameMessage(name *entities.Name) string {
	return fmt.Sprintf(
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
	)
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
		return "🎲 Смешанный режим"
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

	// Основная статистика
	sb.WriteString(md(fmt.Sprintf("✅ Выучено: %d/99 (%.1f%%)\n",
		summary.Learned, summary.Percentage)))

	sb.WriteString(md(fmt.Sprintf("📚 В процессе: %d/99\n", summary.InProgress)))

	// Детализация "В процессе"
	if summary.InProgress > 0 {
		sb.WriteString(md(fmt.Sprintf("  ├─ 🆕 Новые: %d\n", summary.NewCount)))
		sb.WriteString(md(fmt.Sprintf("  └─ 📖 Изучаются: %d\n", summary.LearningCount)))
	}

	sb.WriteString(md(fmt.Sprintf("⭕ Не начато: %d/99\n", summary.NotStarted)))

	sb.WriteString("\n")

	// SRS информация
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
func buildReminderSettingsMessage(reminder *entities.UserReminders) string {
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
			"\n%s %s\n%s %s — %s",
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

	// Заголовок с контекстом
	if payload.Stats.DueToday > 0 {
		sb.WriteString(md("🔔 "))
		sb.WriteString(bold("Время повторить имена Аллаха!"))
		sb.WriteString("\n\n")
		sb.WriteString(md("📖 Имя для повторения:"))
	} else {
		sb.WriteString(md("🌟 "))
		sb.WriteString(bold("Время узнать новое имя Аллаха!"))
		sb.WriteString("\n\n")
		sb.WriteString(md("📖 Имя на сегодня:"))
	}
	sb.WriteString("\n\n")

	// Часть 1: Карточка имени
	sb.WriteString(formatNameMessage(&payload.Name))
	sb.WriteString("\n\n")

	// Часть 2: Разделитель
	sb.WriteString("━━━━━━━━━━━━━━━━\n")

	// Часть 3: Статистика
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

// buildQuizQuestionText formats quiz question text from database question.
func buildQuizQuestionText(question *entities.QuizQuestion, name *entities.Name, currentNum, totalQuestions int) string {
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

	return fmt.Sprintf(
		"%s\n\n%s",
		md(fmt.Sprintf("Вопрос %d из %d", currentNum, totalQuestions)),
		bold(questionPrompt),
	)
}
