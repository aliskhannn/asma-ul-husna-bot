package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

func (h *Handler) numberHandler(numStr string, userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		msg := newHTMLMessage(chatID, "")

		n, err := strconv.Atoi(numStr)
		if err != nil {
			msg.Text = msgIncorrectNameNumber
			return h.send(msg)
		}

		if n < 1 || n > 99 {
			msg.Text = msgOutOfRangeNumber
			return h.send(msg)
		}

		msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (*entities.Name, error) {
			return h.nameService.GetByNumber(ctx, n)
		}, chatID)
		if err != nil {
			return err
		}

		err = h.send(msg)
		if err != nil {
			return err
		}

		if audio != nil {
			_ = h.send(*audio)
		}

		if err = h.progressService.MarkAsViewed(ctx, userID, n); err != nil {
			return err
		}

		return nil
	}
}

func (h *Handler) randomHandler(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		name, err := h.nameService.GetRandom(ctx)
		if err != nil {
			return err
		}

		msg := newHTMLMessage(chatID, formatNameMessage(name))
		if err = h.send(msg); err != nil {
			return err
		}

		if name.Audio != "" {
			audio := buildNameAudio(name, chatID)
			if err = h.send(*audio); err != nil {
				return err
			}
		}

		if err = h.progressService.MarkAsViewed(ctx, userID, name.Number); err != nil {
			return err
		}

		return nil
	}
}

func (h *Handler) allCommandHandler(ctx context.Context, chatID int64) error {
	msg := newHTMLMessage(chatID, "")

	names, err := h.getAllNames(ctx)
	if err != nil {
		return err
	}
	if names == nil {
		msg.Text = msgNameUnavailable
		return h.send(msg)
	}

	page := 0
	text, totalPages := buildNamesPage(names, page)

	prevData := fmt.Sprintf("name:%d", page-1)
	nextData := fmt.Sprintf("name:%d", page+1)

	msg.Text = text
	kb := buildNameKeyboard(page, totalPages, prevData, nextData)
	if kb != nil {
		msg.ReplyMarkup = *kb
	}

	return h.send(msg)
}

func (h *Handler) rangeCommandHandler(argsStr string) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		args := strings.Fields(argsStr)
		if len(args) != 2 {
			return h.send(newHTMLMessage(chatID, msgUseRange))
		}

		from, errFrom := strconv.Atoi(args[0])
		to, errTo := strconv.Atoi(args[1])
		if errFrom != nil || errTo != nil || from < 1 || to > 99 || from > to {
			return h.send(newHTMLMessage(chatID, msgInvalidRange))
		}

		names, err := h.getAllNames(ctx)
		if err != nil {
			return err
		}
		if names == nil {
			return h.send(newHTMLMessage(chatID, msgNameUnavailable))
		}

		pages := buildRangePages(names, from, to)
		if len(pages) == 0 {
			return h.send(newHTMLMessage(chatID, msgNameUnavailable))
		}

		page := 0
		totalPages := len(pages)

		prevData := fmt.Sprintf("range:%d:%d:%d", page-1, from, to)
		nextData := fmt.Sprintf("range:%d:%d:%d", page+1, from, to)

		msg := newHTMLMessage(chatID, pages[page])
		kb := buildNameKeyboard(page, totalPages, prevData, nextData)
		if kb != nil {
			msg.ReplyMarkup = *kb
		}

		return h.send(msg)
	}
}

func (h *Handler) progressHandler(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		msg := newHTMLMessage(userID, "")

		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			msg.Text = msgSettingsUnavailable
			return h.send(msg)
		}

		summary, err := h.progressService.GetProgressSummary(ctx, userID, settings.NamesPerDay)
		if err != nil {
			msg.Text = msgProgressUnavailable
			return h.send(msg)
		}

		progressBar := buildProgressBar(summary.Learned, 99, 20)

		text := fmt.Sprintf(
			"<b>📊 Ваш прогресс</b>\n\n"+
				"%s\n\n"+
				"✅ <b>Выучено:</b> %d / 99 (%.1f%%)\n"+
				"📖 <b>В процессе:</b> %d\n"+
				"⏳ <b>Не начато:</b> %d\n\n"+
				"🎯 <b>Точность:</b> %.1f%%\n"+
				"📅 <b>Имён в день:</b> %d\n"+
				"⏰ <b>Дней до завершения:</b> %d\n",
			progressBar,
			summary.Learned,
			summary.Percentage,
			summary.InProgress,
			summary.NotStarted,
			summary.Accuracy,
			settings.NamesPerDay,
			summary.DaysToComplete,
		)

		msg.Text = text
		return h.send(msg)
	}
}

func (h *Handler) settingsHandler(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		msg := newHTMLMessage(chatID, "")

		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			msg.Text = msgSettingsUnavailable
			return h.send(msg)
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
		return h.send(msg)
	}
}
