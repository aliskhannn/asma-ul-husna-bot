package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	switch {
	case strings.HasPrefix(cb.Data, "range:"):
		h.withCallbackErrorHandling(h.rangeCallbackHandler)(ctx, cb)
	case strings.HasPrefix(cb.Data, "name:"):
		h.withCallbackErrorHandling(h.allCallbackHandler)(ctx, cb)
	default:
		h.logger.Warn("unknown callback data prefix",
			zap.String("data", cb.Data),
		)
		return
	}

	// Remove the user's "clock".
	answer := tgbotapi.NewCallback(cb.ID, "")
	if _, err := h.bot.Request(answer); err != nil {
		h.logger.Error("callback answer error",
			zap.Error(err),
			zap.String("data", cb.Data),
		)
	}
}

func (h *Handler) allCallbackHandler(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	pageStr := strings.TrimPrefix(cb.Data, "name:")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 0 {
		h.logger.Warn("invalid page in callback",
			zap.String("data", cb.Data),
			zap.Error(err),
		)
		return nil
	}

	names, err := h.getAllNames(ctx)
	if err != nil {
		return err
	}
	if names == nil {
		msg := newHTMLMessage(cb.Message.Chat.ID, msgNameUnavailable)
		h.send(msg)
		return nil
	}

	text, totalPages := buildNamesPage(names, page)
	if totalPages == 0 || page >= totalPages {
		h.logger.Warn("page out of range",
			zap.Int("page", page),
			zap.Int("total_pages", totalPages),
		)
		return nil
	}

	prevData := fmt.Sprintf("name:%d", page-1)
	nextData := fmt.Sprintf("name:%d", page+1)

	kb := buildNameKeyboard(page, totalPages, prevData, nextData)

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		edit.ReplyMarkup = kb
	}

	h.send(edit)
	return nil
}

func (h *Handler) rangeCallbackHandler(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 4 {
		h.logger.Warn("invalid range callback data",
			zap.String("data", cb.Data),
		)
		return nil
	}

	page, err1 := strconv.Atoi(parts[1])
	from, err2 := strconv.Atoi(parts[2])
	to, err3 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil || err3 != nil || page < 0 || from < 1 || to > 99 || from > to {
		h.logger.Warn("invalid range callback values",
			zap.String("data", cb.Data),
			zap.Error(err1),
			zap.Error(err2),
			zap.Error(err3),
		)
		return nil
	}

	names, err := h.getAllNames(ctx)
	if err != nil {
		return err
	}
	if names == nil {
		msg := newHTMLMessage(cb.Message.Chat.ID, msgNameUnavailable)
		h.send(msg)
		return nil
	}

	pages := buildRangePages(names, from, to)
	totalPages := len(pages)
	if totalPages == 0 || page >= totalPages {
		h.logger.Warn("range page out of range",
			zap.Int("page", page),
			zap.Int("total_pages", totalPages),
			zap.Int("from", from),
			zap.Int("to", to),
		)
		return nil
	}

	text := pages[page]

	prevData := fmt.Sprintf("range:%d:%d:%d", page-1, from, to)
	nextData := fmt.Sprintf("range:%d:%d:%d", page+1, from, to)

	kb := buildNameKeyboard(page, totalPages, prevData, nextData)

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		edit.ReplyMarkup = kb
	}

	h.send(edit)
	return nil
}
