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
)

// Error messages.
var (
	msgIncorrectNameNumber = "Некорректный ввод. Введите число от 1 до 99."
	msgOutOfRangeNumber    = "Номер имени должен быть от 1 до 99."
	msgUseRange            = "Используйте: /range 25 30"
	msgInvalidRange        = "Некорректный диапазон. Пример: /range 25 30"
	msgNameUnavailable     = "Не удалось получить имя. Попробуйте позже."
	msgProgressUnavailable = "Не удалось получить прогресс. Попробуйте позже."
	msgSettingsUnavailable = "Не удалось получить настройки. Попробуйте позже."
	msgQuizUnavailable     = "Не удалось создать квиз, попробуйте позже."
	msgInternalError       = "Что‑то пошло не так. Попробуйте позже."
	msgUnknownCommand      = "Неизвестная команда. Список доступных команд:\n\n/all — посмотреть все имена\n/random — получить случайное имя\n/range N M — посмотреть имена с N по M"
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

// WelcomeMarkdownV2 builds welcome message safely for MarkdownV2.
func WelcomeMarkdownV2() string {
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

	sb.WriteString(md("Чтобы начать:"))
	sb.WriteString("\n\n")

	// EscapeText will escape dots in "1." etc. automatically. [page:0]
	sb.WriteString(md("1. Введите 1 для просмотра первого имени."))
	sb.WriteString("\n")
	sb.WriteString(md("2. Используйте /random чтобы получить рандомное имя."))
	sb.WriteString("\n")
	sb.WriteString(md("3. Нажмите /all для просмотра всех имён."))
	sb.WriteString("\n")
	sb.WriteString(md("4. Используйте /range N M для просмотра имён с N по M."))
	sb.WriteString("\n")
	sb.WriteString(md("5. Нажмите /settings для выбора языка и настройки напоминаний."))
	sb.WriteString("\n")
	sb.WriteString(md("6. Нажмите /help для получения помощи."))
	sb.WriteString("\n\n")

	sb.WriteString(md("Начните свой путь к знанию прямо сейчас!"))

	return sb.String()
}

// formatNameMessage formats a single name message (MarkdownV2 safe).
func formatNameMessage(name *entities.Name) string {
	// Everything coming from DB/service must be escaped.
	// Markup is added around escaped text. [page:0]
	return fmt.Sprintf(
		"%s%s %s\n\n%s %s\n%s %s\n\n%s %s",
		lrm,
		bold(fmt.Sprintf("%d.", name.Number)),
		md(name.ArabicName),

		bold("Транслитерация:"),
		md(name.Transliteration),

		bold("Перевод:"),
		md(name.Translation),

		bold("Значение:"),
		md(name.Meaning),
	)
}

// buildNameResponse builds name message and optional audio.
func buildNameResponse(
	ctx context.Context,
	get func(ctx2 context.Context) (*entities.Name, error),
	chatID int64,
) (tgbotapi.MessageConfig, *tgbotapi.AudioConfig, error) {
	msg := newMessage(chatID, "")

	name, err := get(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			msg.Text = msgIncorrectNameNumber
			return msg, nil, nil
		}

		if errors.Is(err, repository.ErrRepositoryEmpty) {
			msg.Text = msgNameUnavailable
			return msg, nil, nil
		}

		msg.Text = msgNameUnavailable
		return msg, nil, err
	}

	msg.Text = formatNameMessage(name)

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
		md("🎯 Квиз начинается!"),
		bold("Режим:"),
		md(modeText),
		md("Выберите правильный вариант ответа для каждого вопроса."),
	)
}

// formatQuizMode formats quiz mode for display.
func formatQuizMode(mode string) string {
	switch mode {
	case "new_only":
		return "🆕 Только новые"
	case "review_only":
		return "🔄 Только повторение"
	case "mixed":
		return "🎲 Смешанный режим"
	case "daily":
		return "📅 Ежедневный"
	default:
		return mode
	}
}

// formatBool formats boolean as enabled/disabled.
func formatBool(b bool) string {
	if b {
		return "Включено ✅"
	}
	return "Выключено ❌"
}

// formatQuizQuestion formats a quiz question (MarkdownV2 safe for question text).
func formatQuizQuestion(q *entities.Question, currentNum, totalQuestions int) string {
	return fmt.Sprintf(
		"%s\n\n%s",
		md(fmt.Sprintf("Вопрос %d из %d", currentNum, totalQuestions)),
		md(q.Question),
	)
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
		bold("Результат:"),
		md(fmt.Sprintf("%d/%d (%.0f%%)", session.CorrectAnswers, session.TotalQuestions, percentage)),
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
