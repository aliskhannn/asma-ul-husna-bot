package telegram

import (
	"context"

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
			h.sendError(chatID, msgInternalError)
			return nil
		}
		return nil
	}
}
