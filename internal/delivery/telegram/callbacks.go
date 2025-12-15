package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
