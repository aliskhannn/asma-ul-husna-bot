package telegram

import (
	"context"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/entities"
)

type NameUseCase interface {
	GetNameByNumber(ctx context.Context, number int) (entities.Name, error)
	GetRandomName(ctx context.Context) (entities.Name, error)
	GetAllNames(ctx context.Context) ([]entities.Name, error)
}

type UserUseCase interface {
	EnsureUser(ctx context.Context, userID int64, firstName, lastName string, username string, languageCode string) error
}

type Handler struct {
	bot         *tgbotapi.BotAPI
	nameUseCase NameUseCase
	userUseCase UserUseCase
}

func NewHandler(bot *tgbotapi.BotAPI, nameUseCase NameUseCase, userUseCase UserUseCase) *Handler {
	return &Handler{
		bot:         bot,
		nameUseCase: nameUseCase,
		userUseCase: userUseCase,
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
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	user := update.Message.From
	err := h.userUseCase.EnsureUser(
		ctx,
		user.ID,
		user.FirstName,
		user.LastName,
		user.UserName,
		user.LanguageCode,
	)
	if err != nil {
		log.Println(err)
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
			msg, audio := buildNameResponse(ctx, h.nameUseCase.GetRandomName, chatID)
			h.send(msg)
			if audio != nil {
				h.send(*audio)
			}

		case "all": // TODO: refactor
			names, err := h.nameUseCase.GetAllNames(ctx)
			if err != nil || len(names) == 0 {
				msg.Text = msgFailedToGetName
				h.send(msg)
				return
			}

			idx := 0
			name := names[idx]

			text := processName(name)
			msg.Text = text
			msg.ReplyMarkup = buildNameKeyboard(idx, len(names))
			h.send(msg)

		default:
			msg.Text = msgUnknownCommand
			h.send(msg)
		}

		return
	}

	// TODO: refactor
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

	msg, audio := buildNameResponse(ctx, func(ctx context.Context) (entities.Name, error) {
		return h.nameUseCase.GetNameByNumber(ctx, n)
	}, chatID)

	h.send(msg)
	if audio != nil {
		h.send(*audio)
	}
}

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	data := cb.Data

	if !strings.HasPrefix(data, "name:") {
		return
	}

	idxStr := strings.TrimPrefix(data, "name:")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx > 98 {
		log.Printf("invalid idx in callback: %s", data)
		return
	}

	names, err := h.nameUseCase.GetAllNames(ctx)
	if err != nil {
		log.Printf("failed to get all names: %v", err)
		return
	}
	if idx >= len(names) {
		log.Printf("idx out of range: %d (len=%d)", idx, len(names))
		return
	}

	name := names[idx]
	text := processName(name)

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	kb := buildNameKeyboard(idx, len(names))
	edit.ReplyMarkup = &kb

	h.send(edit)

	// Remove the user's "clock".
	answer := tgbotapi.NewCallback(cb.ID, "")
	if _, err := h.bot.Request(answer); err != nil {
		log.Println("callback answer error:", err)
	}
}

func (h *Handler) send(c tgbotapi.Chattable) {
	if _, err := h.bot.Send(c); err != nil {
		log.Printf("failed to send telegram message: %v", err)
	}
}
