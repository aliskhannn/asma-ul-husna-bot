package telegram

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type HandlerFunc func(ctx context.Context, chatID int64) error

func (h *Handler) withErrorHandling(fn HandlerFunc) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		if err := fn(ctx, chatID); err != nil {
			h.logger.Error("handle error",
				zap.Int64("chat_id", chatID),
				zap.Error(err),
			)
			msg := newPlainMessage(chatID, msgInternalError)
			return h.send(msg)
		}
		return nil
	}
}

type CallbackHandlerFunc func(ctx context.Context, cb *tgbotapi.CallbackQuery) error

func (h *Handler) withCallbackErrorHandling(fn CallbackHandlerFunc) func(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	return func(ctx context.Context, cb *tgbotapi.CallbackQuery) {
		if err := fn(ctx, cb); err != nil {
			h.logger.Error("callback handler error",
				zap.Error(err),
				zap.String("data", cb.Data),
				zap.Int64("user_id", cb.From.ID),
			)
			if cb.Message != nil {
				_ = h.send(newPlainMessage(cb.Message.Chat.ID, msgInternalError))
			}
		}
	}
}
