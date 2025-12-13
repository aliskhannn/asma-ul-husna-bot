package telegram

import (
	"context"
	"log"
	"path/filepath"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

type NameUseCase interface {
	GetNameByNumber(ctx context.Context, number int) entities.Name
	GetRandomName(ctx context.Context) entities.Name
	GetAllNames(ctx context.Context) []entities.Name
}

type Handler struct {
	bot         *tgbotapi.BotAPI
	nameUseCase NameUseCase
}

func NewHandler(bot *tgbotapi.BotAPI, nameUseCase NameUseCase) *Handler {
	return &Handler{
		bot:         bot,
		nameUseCase: nameUseCase,
	}
}

func (h *Handler) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			h.handleUpdate(ctx, update)
		}
	}
}

func (h *Handler) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "")
	msg.ParseMode = tgbotapi.ModeHTML

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			msg.Text = msgWelcome
			h.send(msg)

		case "random":
			msg, audio := h.buildNameResponse(ctx, h.nameUseCase.GetRandomName, chatID)
			h.send(msg)
			if audio != nil {
				h.send(*audio)
			}

		default:
			msg.Text = msgUnknownCommand
			h.send(msg)
		}
	} else {
		n, err := strconv.Atoi(update.Message.Text)
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

		msg, audio := h.buildNameResponse(ctx, func(ctx context.Context) entities.Name {
			return h.nameUseCase.GetNameByNumber(ctx, n)
		}, chatID)

		h.send(msg)
		if audio != nil {
			h.send(*audio)
		}
	}
}

func (h *Handler) buildNameResponse(
	ctx context.Context,
	get func(ctx2 context.Context) entities.Name, chatID int64,
) (tgbotapi.MessageConfig, *tgbotapi.AudioConfig) {
	msg := tgbotapi.NewMessage(chatID, "")
	msg.ParseMode = tgbotapi.ModeHTML

	name := get(ctx)

	msg.Text = formatName(name)

	if name.Audio == "" {
		return msg, nil
	}

	audio := h.newAudio(name, chatID)
	return msg, audio
}

func (h *Handler) newAudio(name entities.Name, chatID int64) *tgbotapi.AudioConfig {
	path := filepath.Join("assets", "audio", name.Audio)

	a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(path))
	a.Caption = name.Transliteration

	return &a
}

func (h *Handler) send(c tgbotapi.Chattable) {
	if _, err := h.bot.Send(c); err != nil {
		log.Printf("failed to send telegram message: %v", err)
	}
}
