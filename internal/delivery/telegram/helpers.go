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

var msgWelcome = `السلام عليكم ورحمة الله وبركاته

Абу Хурайра, да будет доволен им Аллах, передаёт, что Посланник Аллаха ﷺ сказал: «Поистине, у Аллаха девяносто девять имён — сотня без одного, и каждый, кто запомнит их, войдёт в Рай. Поистине, Он (— это Тот, Кто) не имеет пары /витр/, и Он любит (всё) непарное». (Аль-Бухари, 6410)

<b>Asma ul Husna Bot</b> поможет вам в изучении <b>99 имён Алла́ха</b> (асма̄'у -лла̄һи ль-х̣усна̄ — «прекраснейшие имена Аллаха»).

С нами вы сможете:

📖 Изучать каждое имя с <b>переводом</b>, <b>транслитерацией</b> и <b>аудиопроизношением</b>.
⏰ Настроить <b>гибкие напоминания</b> для ежедневного повторения.
🧠 Проходить <b>квизы</b> для проверки и отслеживания прогресса.

Чтобы начать:

1. Введите 1 для просмотра первого имени.
2. Используйте /random чтобы получить рандомное имя.
3. Нажмите /all для просмотра всех имён.
4. Используйте /range N M для просмотра имён с N по M.
5. Нажмите /settings для выбора языка и настройки напоминаний.
6. Нажмите /help для получения помощи.

<b>Начните свой путь к знанию прямо сейчас!</b>`

var (
	msgIncorrectNameNumber = "Некорректный ввод. Введите число от 1 до 99."
	msgOutOfRangeNumber    = "Номер имени должен быть от 1 до 99."
	msgUseRange            = "Используйте: /range 25 30"
	msgInvalidRange        = "Некорректный диапазон. Пример: /range 25 30"

	msgNameUnavailable     = "Не удалось получить имя. Попробуйте позже."
	msgProgressUnavailable = "Не удалось получить прогресс. Попробуйте позже."
	msgSettingsUnavailable = "Не удалось получить настройки. Попробуйте позже."
	msgInternalError       = "Что‑то пошло не так. Попробуйте позже."

	msgUnknownCommand = "Неизвестная команда. Список доступных команд:\n\n/all — посмотреть все имена\n/random — получить случайное имя\n/range N M — посмотреть имена с N по M"
)

const (
	lrm          = "\u200E"
	namesPerPage = 5
)

func formatNameMessage(name *entities.Name) string {
	return fmt.Sprintf(
		"%s<b>%d. </b>%s<b>\n\n"+
			"Транслитерация:</b>  %s\n"+
			"<b>Перевод:</b> %s\n\n"+
			"<b>Значение:</b> %s",
		lrm,
		name.Number,
		name.ArabicName,
		name.Transliteration,
		name.Translation,
		name.Meaning,
	)
}

func buildNameResponse(
	ctx context.Context,
	get func(ctx2 context.Context) (*entities.Name, error), chatID int64,
) (tgbotapi.MessageConfig, *tgbotapi.AudioConfig, error) {
	msg := newHTMLMessage(chatID, "")

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

func buildNameAudio(name *entities.Name, chatID int64) *tgbotapi.AudioConfig {
	path := filepath.Join("assets", "audio", name.Audio)

	a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(path))
	a.Caption = name.Transliteration

	return &a
}

func newHTMLMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	return msg
}

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
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔔 Напоминания", "reminder_settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад", "main_menu"),
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
