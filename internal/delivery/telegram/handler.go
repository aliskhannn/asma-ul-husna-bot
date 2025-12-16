package telegram

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

type UserService interface {
	EnsureUser(ctx context.Context, userID int64, firstName, lastName string, username string, languageCode string) error
}

type NameService interface {
	GetByNumber(ctx context.Context, number int) (*entities.Name, error)
	GetRandom(ctx context.Context) (*entities.Name, error)
	GetAll(ctx context.Context) ([]*entities.Name, error)
}

type ProgressService interface {
	GetProgressSummary(ctx context.Context, userID int64, namesPerDay int) (*service.ProgressSummary, error)
	MarkAsViewed(ctx context.Context, userID int64, nameNumber int) error
}

type SettingsService interface {
	GetOrCreate(ctx context.Context, userID int64) (*entities.UserSettings, error)
}

type Handler struct {
	bot             *tgbotapi.BotAPI
	logger          *zap.Logger
	nameService     NameService
	userService     UserService
	progressService ProgressService
	settingsService SettingsService
}

func NewHandler(
	bot *tgbotapi.BotAPI,
	logger *zap.Logger,
	nameService NameService,
	userService UserService,
	progressService ProgressService,
	settingsService SettingsService,
) *Handler {
	return &Handler{
		bot:             bot,
		logger:          logger,
		nameService:     nameService,
		userService:     userService,
		progressService: progressService,
		settingsService: settingsService,
	}
}

func (h *Handler) Run(ctx context.Context) error {
	h.logger.Info("telegram handler started")
	defer h.logger.Info("telegram handler stopped")

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
		h.logger.Debug("callback received",
			zap.Int64("user_id", update.CallbackQuery.From.ID),
			zap.String("data", update.CallbackQuery.Data),
		)
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		h.logger.Debug("update without message and callback")
		return
	}

	h.logger.Debug("update received",
		zap.Int64("chat_id", update.Message.Chat.ID),
		zap.String("text", update.Message.Text),
	)

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
		h.logger.Error("failed to ensure user",
			zap.Int64("user_id", from.ID),
			zap.Error(err),
		)
	}

	chatID := update.Message.Chat.ID
	msg := newHTMLMessage(chatID, "")

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			msg.Text = msgWelcome
			h.send(msg)

		case "random":
			_ = h.withErrorHandling(h.randomHandler(from.ID))(ctx, chatID)

		case "all":
			h.handleAllCommand(ctx, chatID)

		case "range":
			h.handleRangeCommand(ctx, chatID, update.Message.CommandArguments())

		case "progress":
			_ = h.withErrorHandling(h.progressHandler(from.ID))(ctx, chatID)

		case "settings":
			_ = h.withErrorHandling(h.settingsHandler(from.ID))(ctx, chatID)

		default:
			msg.Text = msgUnknownCommand
			h.send(msg)
		}

		return
	}

	_ = h.withErrorHandling(h.numberHandler(update.Message.Text, from.ID))(ctx, chatID)
}

func (h *Handler) sendError(chatID int64, err string) {
	msg := newHTMLMessage(chatID, err)
	h.send(msg)
}

func (h *Handler) send(c tgbotapi.Chattable) {
	if _, err := h.bot.Send(c); err != nil {
		h.logger.Error("failed to send telegram message",
			zap.Error(err),
		)
	}
}
