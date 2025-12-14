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
	msg := newHTMLMessage(chatID, "")

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

		case "all":
			names, err := h.nameUseCase.GetAllNames(ctx)
			if err != nil || len(names) == 0 {
				msg.Text = msgFailedToGetName
				h.send(msg)
				return
			}

			page := 0
			text, totalPages := buildNamesPage(names, page)

			msg.Text = text
			msg.ReplyMarkup = buildNameKeyboard(page, totalPages)
			h.send(msg)

		default:
			msg.Text = msgUnknownCommand
			h.send(msg)
		}

		return
	}

	h.handleNumberInput(ctx, chatID, update.Message.Text)
}

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	data := cb.Data

	if !strings.HasPrefix(data, "name:") {
		return
	}

	pageStr := strings.TrimPrefix(data, "name:")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 0 {
		log.Printf("invalid page in callback: %s", data)
		return
	}

	names, err := h.nameUseCase.GetAllNames(ctx)
	if err != nil {
		log.Printf("failed to get all names: %v", err)
		return
	}

	text, totalPages := buildNamesPage(names, page)
	if totalPages == 0 || page >= totalPages {
		log.Printf("page out of range: %d (totalPages=%d)", page, totalPages)
		return
	}

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	kb := buildNameKeyboard(page, totalPages)
	edit.ReplyMarkup = &kb

	h.send(edit)

	// Remove the user's "clock".
	answer := tgbotapi.NewCallback(cb.ID, "")
	if _, err := h.bot.Request(answer); err != nil {
		log.Println("callback answer error:", err)
	}
}

func (h *Handler) handleNumberInput(ctx context.Context, chatID int64, text string) {
	msg := newHTMLMessage(chatID, "")

	n, err := strconv.Atoi(text)
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

func (h *Handler) send(c tgbotapi.Chattable) {
	if _, err := h.bot.Send(c); err != nil {
		log.Printf("failed to send telegram message: %v", err)
	}
}
