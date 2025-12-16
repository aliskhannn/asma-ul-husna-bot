package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

func (h *Handler) numberHandler(numStr string) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		msg := newHTMLMessage(chatID, "")

		n, err := strconv.Atoi(numStr)
		if err != nil {
			msg.Text = msgIncorrectNameNumber
			h.send(msg)
			return nil
		}

		if n < 1 || n > 99 {
			h.sendError(chatID, msgOutOfRangeNumber)
			return nil
		}

		msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (entities.Name, error) {
			return h.nameService.GetByNumber(ctx, n)
		}, chatID)
		if err != nil {
			return err
		}

		h.send(msg)
		if audio != nil {
			h.send(*audio)
		}
		return nil
	}
}

func (h *Handler) randomHandler() HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		msg, audio, err := buildNameResponse(ctx, h.nameService.GetRandom, chatID)
		if err != nil {
			return err
		}

		h.send(msg)
		if audio != nil {
			h.send(*audio)
		}
		return nil
	}
}

func (h *Handler) handleAllCommand(ctx context.Context, chatID int64) {
	msg := newHTMLMessage(chatID, "")

	names := h.getAllNames(ctx)
	if names == nil {
		msg.Text = msgNameUnavailable
		h.send(msg)
		return
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

	h.send(msg)
}

func (h *Handler) handleRangeCommand(ctx context.Context, chatID int64, argsStr string) {
	msg := newHTMLMessage(chatID, "")

	args := strings.Fields(argsStr)
	if len(args) != 2 {
		msg.Text = msgUseRange
		h.send(msg)
		return
	}

	from, errFrom := strconv.Atoi(args[0])
	to, errTo := strconv.Atoi(args[1])
	if errFrom != nil || errTo != nil || from < 1 || to > 99 || from > to {
		msg.Text = msgInvalidRange
		h.send(msg)
		return
	}

	names := h.getAllNames(ctx)
	if names == nil {
		msg.Text = msgNameUnavailable
		h.send(msg)
		return
	}

	pages := buildRangePages(names, from, to)
	if len(pages) == 0 {
		msg.Text = msgNameUnavailable
		h.send(msg)
		return
	}

	page := 0
	totalPages := len(pages)

	prevData := fmt.Sprintf("range:%d:%d:%d", page-1, from, to)
	nextData := fmt.Sprintf("range:%d:%d:%d", page+1, from, to)

	msg = newHTMLMessage(chatID, pages[page])
	kb := buildNameKeyboard(page, totalPages, prevData, nextData)
	if kb != nil {
		msg.ReplyMarkup = *kb
	}

	h.send(msg)
}

func (h *Handler) progressHandler(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		msg := newHTMLMessage(userID, "")

		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			msg.Text = msgSettingsUnavailable
			h.send(msg)
			return err
		}

		summary, err := h.progressService.GetProgressSummary(ctx, userID, settings.NamesPerDay)
		if err != nil {
			msg.Text = msgProgressUnavailable
			h.send(msg)
			return err
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
		h.send(msg)
		return nil
	}
}

func (h *Handler) settingsHandler(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		msg := newHTMLMessage(chatID, "")

		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			msg.Text = msgSettingsUnavailable
			h.send(msg)
			return err
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
		h.send(msg)

		return nil
	}
}
