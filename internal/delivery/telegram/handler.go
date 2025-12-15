package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type NameService interface {
	GetNameByNumber(ctx context.Context, number int) (entities.Name, error)
	GetRandomName(ctx context.Context) (entities.Name, error)
	GetAllNames(ctx context.Context) ([]entities.Name, error)
}

type UserUseCase interface {
	EnsureUser(ctx context.Context, userID int64, firstName, lastName string, username string, languageCode string) error
}

type Handler struct {
	bot         *tgbotapi.BotAPI
	nameService NameService
	userUseCase UserUseCase
}

func NewHandler(bot *tgbotapi.BotAPI, nameService NameService, userUseCase UserUseCase) *Handler {
	return &Handler{
		bot:         bot,
		nameService: nameService,
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
			msg, audio := buildNameResponse(ctx, h.nameService.GetRandomName, chatID)
			h.send(msg)
			if audio != nil {
				h.send(*audio)
			}

		case "all":
			names := h.getAllNames(ctx)
			if names == nil {
				msg.Text = msgFailedToGetName
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

		case "range":
			args := strings.Fields(update.Message.CommandArguments())
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
				msg.Text = msgFailedToGetName
				h.send(msg)
				return
			}

			pages := buildRangePages(names, from, to)
			if len(pages) == 0 {
				msg.Text = msgFailedToGetName
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

		default:
			msg.Text = msgUnknownCommand
			h.send(msg)
		}

		return
	}

	h.handleNumberInput(ctx, chatID, update.Message.Text)
}

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	var (
		text string
		kb   *tgbotapi.InlineKeyboardMarkup
		ok   bool
	)

	switch {
	case strings.HasPrefix(cb.Data, "range:"):
		text, kb, ok = h.handleRangeCallback(ctx, cb)
	case strings.HasPrefix(cb.Data, "name:"):
		text, kb, ok = h.handleAllCallback(ctx, cb)
	default:
		return
	}

	if !ok {
		return
	}

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		edit.ReplyMarkup = kb
	}

	h.send(edit)

	// Remove the user's "clock".
	answer := tgbotapi.NewCallback(cb.ID, "")
	if _, err := h.bot.Request(answer); err != nil {
		log.Println("callback answer error:", err)
	}
}

func (h *Handler) handleAllCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) (string, *tgbotapi.InlineKeyboardMarkup, bool) {
	pageStr := strings.TrimPrefix(cb.Data, "name:")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 0 {
		log.Printf("invalid page in callback: %s", cb.Data)
		return "", nil, false
	}

	names := h.getAllNames(ctx)
	if names == nil {
		return "", nil, false
	}

	text, totalPages := buildNamesPage(names, page)
	if totalPages == 0 || page >= totalPages {
		log.Printf("page out of range: %d (totalPages=%d)", page, totalPages)
		return "", nil, false
	}

	prevData := fmt.Sprintf("name:%d", page-1)
	nextData := fmt.Sprintf("name:%d", page+1)

	kb := buildNameKeyboard(page, totalPages, prevData, nextData)

	return text, kb, true
}

func (h *Handler) handleRangeCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) (string, *tgbotapi.InlineKeyboardMarkup, bool) {
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 4 {
		log.Printf("invalid range callback data: %s", cb.Data)
		return "", nil, false
	}

	page, err1 := strconv.Atoi(parts[1])
	from, err2 := strconv.Atoi(parts[2])
	to, err3 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil || err3 != nil || page < 0 || from < 1 || to > 99 || from > to {
		log.Printf("invalid range callback values: %s", cb.Data)
		return "", nil, false
	}

	names := h.getAllNames(ctx)
	if names == nil {
		return "", nil, false
	}

	pages := buildRangePages(names, from, to)
	totalPages := len(pages)
	if totalPages == 0 || page >= totalPages {
		log.Printf("range page out of range: %d (totalPages=%d)", page, totalPages)
		return "", nil, false
	}

	text := pages[page]

	prevData := fmt.Sprintf("range:%d:%d:%d", page-1, from, to)
	nextData := fmt.Sprintf("range:%d:%d:%d", page+1, from, to)

	kb := buildNameKeyboard(page, totalPages, prevData, nextData)

	return text, kb, true
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
		return h.nameService.GetNameByNumber(ctx, n)
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

func (h *Handler) getAllNames(ctx context.Context) []entities.Name {
	names, err := h.nameService.GetAllNames(ctx)
	if err != nil {
		log.Printf("failed to get all names: %v", err)
		return nil
	}
	if len(names) == 0 {
		log.Println("no names found")
		return nil
	}

	return names
}
