package telegram

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

var (
	msgWelcome = `<b>السلام عليكم ورحمة الله وبركاته</b>

Абу Хурайра, да будет доволен им Аллах, передаёт, что Посланник Аллаха ﷺ сказал: «Поистине, у Аллаха девяносто девять имён — сотня без одного, и каждый, кто запомнит их, войдёт в Рай. Поистине, Он (— это Тот, Кто) не имеет пары /витр/, и Он любит (всё) непарное». (Аль-Бухари, 6410)

<b>Asma ul Husna Bot</b> поможет вам в изучении <b>99 имён Алла́ха</b> (асма̄'у -лла̄һи ль-х̣усна̄ — «прекраснейшие имена Аллаха»).

С нами вы сможете:

📖 Изучать каждое имя с <b>переводом</b>, <b>транслитерацией</b> и <b>аудиопроизношением</b>.
⏰ Настроить <b>гибкие напоминания</b> для ежедневного повторения.
🧠 Проходить <b>квизы</b> для проверки и отслеживания прогресса.
🌍 Получать информацию на <b>75 языках</b>.

Чтобы начать:

1. Нажмите /settings для выбора языка и настройки напоминаний.
2. Введите 1 для просмотра первого имени.
3. Используйте /random чтобы получить рандомное имя.

<b>Начните свой путь к знанию прямо сейчас!</b>`
	msgIncorrectNameNumber = "Некорректный ввод. Введите число от 1 до 99."
	msgOutOfRangeNumber    = "Номер имени должен быть от 1 до 99."
	msgFailedToGetName     = "Не удалось получить имя. Попробуйте ещё раз позже."
	msgUnknownCommand      = "Неизвестная команда. Введите номер имени или используйте /random или /help."
)

var (
	prevButtonData = "◀️ Назад"
	nextButtonData = "Вперёд ▶"
)

const lrm = "\u200E"
const perPage = 5

func processName(n entities.Name) string {
	return fmt.Sprintf(
		"%s<b>%d. </b>%s\n\n<b>Транслитерация:</b>  %s\n<b>Перевод:</b> %s\n<b>Значение:</b> %s",
		lrm,
		n.Number,
		n.ArabicName,
		n.Transliteration,
		n.Translation,
		n.Meaning,
	)
}

func buildNameResponse(
	ctx context.Context,
	get func(ctx2 context.Context) (entities.Name, error), chatID int64,
) (tgbotapi.MessageConfig, *tgbotapi.AudioConfig) {
	msg := newHTMLMessage(chatID, "")

	name, err := get(ctx)
	if err != nil {
		msg.Text = msgFailedToGetName
		return msg, nil
	}

	msg.Text = processName(name)

	if name.Audio == "" {
		return msg, nil
	}

	audio := buildNameAudio(name, chatID)
	return msg, audio
}

func buildNameKeyboard(page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	prevData := fmt.Sprintf("name:%d", page-1)
	nextData := fmt.Sprintf("name:%d", page+1)

	var buttons [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	if page > 0 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(prevButtonData, prevData))
	}
	if page < totalPages-1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(nextButtonData, nextData))
	}

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

func buildNameAudio(name entities.Name, chatID int64) *tgbotapi.AudioConfig {
	path := filepath.Join("assets", "audio", name.Audio)

	a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(path))
	a.Caption = name.Transliteration

	return &a
}

func buildNamesPage(names []entities.Name, page int) (text string, totalPages int) {
	totalPages = (len(names) + perPage - 1) / perPage
	if totalPages == 0 {
		return "", 0
	}

	pageNames := paginateNames(names, page, perPage)

	var b strings.Builder
	for i, name := range pageNames {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(processName(name))
	}

	return b.String(), totalPages
}

func paginateNames(names []entities.Name, page, perPage int) []entities.Name {
	start := page * perPage
	end := start + perPage

	if start >= len(names) {
		return nil
	}
	if end > len(names) {
		end = len(names)
	}

	return names[start:end]
}

func newHTMLMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	return msg
}
