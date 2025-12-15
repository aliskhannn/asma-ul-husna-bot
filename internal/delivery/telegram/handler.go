package telegram

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot             *tgbotapi.BotAPI
	nameService     NameService
	userService     UserService
	progressService ProgressService
	settingsService SettingsService
}

func NewHandler(
	bot *tgbotapi.BotAPI,
	nameService NameService,
	userService UserService,
	progressService ProgressService,
	settingsService SettingsService,
) *Handler {
	return &Handler{
		bot:             bot,
		nameService:     nameService,
		userService:     userService,
		progressService: progressService,
		settingsService: settingsService,
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

	from := update.Message.From
	err := h.userService.EnsureUser(
		ctx,
		from.ID,
		from.FirstName,
		from.LastName,
		from.UserName,
		from.LanguageCode,
	)
	if err != nil {
		log.Println(err)
	}

	chatID := update.Message.Chat.ID
	msg := newHTMLMessage(chatID, "")

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			msg.Text = msgWelcome
			h.send(msg)

		case "random":
			h.handleRandomCommand(ctx, chatID)

		case "all":
			h.handleAllCommand(ctx, chatID)

		case "range":
			h.handleRangeCommand(ctx, chatID, update.Message.CommandArguments())

		case "progress":
			h.handleProgressCommand(ctx, from.ID)

		case "settings":
			h.handleSettingsCommand(ctx, chatID, from.ID)

		default:
			msg.Text = msgUnknownCommand
			h.send(msg)
		}

		return
	}

	h.handleNumberCommand(ctx, chatID, update.Message.Text)
}

func (h *Handler) send(c tgbotapi.Chattable) {
	if _, err := h.bot.Send(c); err != nil {
		log.Printf("failed to send telegram message: %v", err)
	}
}
