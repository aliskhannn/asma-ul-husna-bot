package telegram

import (
	"context"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

func (h *Handler) handleNumberCommand(ctx context.Context, chatID int64, numStr string) {
	msg := newHTMLMessage(chatID, "")

	n, err := strconv.Atoi(numStr)
	if err != nil {
		msg.Text = msgIncorrectNameNumber
		h.send(msg)
		return
	}

	if n < 1 || n > 99 {
		msg.Text = msgOutOfRangeNumber
		h.send(msg)
		return
	}

	msg, audio := buildNameResponse(ctx, func(ctx context.Context) (entities.Name, error) {
		return h.nameService.GetNameByNumber(ctx, n)
	}, chatID)

	h.send(msg)
	if audio != nil {
		h.send(*audio)
	}
}

func (h *Handler) handleRandomCommand(ctx context.Context, chatID int64) {
	msg, audio := buildNameResponse(ctx, h.nameService.GetRandomName, chatID)
	h.send(msg)
	if audio != nil {
		h.send(*audio)
	}
}

func buildNameResponse(
	ctx context.Context,
	get func(ctx2 context.Context) (entities.Name, error), chatID int64,
) (tgbotapi.MessageConfig, *tgbotapi.AudioConfig) {
	msg := newHTMLMessage(chatID, "")

	name, err := get(ctx)
	if err != nil {
		msg.Text = msgNameUnavailable
		return msg, nil
	}

	msg.Text = processName(name)

	if name.Audio == "" {
		return msg, nil
	}

	audio := buildNameAudio(name, chatID)
	return msg, audio
}
