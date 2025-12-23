// Package telegram provides handlers for Telegram bot updates.

package telegram

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// Handler is responsible for processing Telegram updates and callbacks.
type Handler struct {
	bot              *tgbotapi.BotAPI
	logger           *zap.Logger
	nameService      NameService
	userService      UserService
	progressService  ProgressService
	settingsService  SettingsService
	quizService      QuizService
	quizStorage      QuizStorage
	reminderService  ReminderService
	dailyNameService DailyNameService
}

// NewHandler creates a new Telegram handler with dependencies.
func NewHandler(
	bot *tgbotapi.BotAPI,
	logger *zap.Logger,
	nameService NameService,
	userService UserService,
	progressService ProgressService,
	settingsService SettingsService,
	quizService QuizService,
	quizStorage QuizStorage,
	reminderService ReminderService,
	dailyNameService DailyNameService,
) *Handler {
	return &Handler{
		bot:              bot,
		logger:           logger,
		nameService:      nameService,
		userService:      userService,
		progressService:  progressService,
		settingsService:  settingsService,
		quizService:      quizService,
		quizStorage:      quizStorage,
		reminderService:  reminderService,
		dailyNameService: dailyNameService,
	}
}

// Run starts the handler loop for processing Telegram updates.
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

// handleUpdate processes incoming Telegram update.
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
		update.Message.Chat.ID,
	)
	if err != nil {
		h.logger.Error("failed to ensure user",
			zap.Int64("user_id", from.ID),
			zap.Error(err),
		)
	}

	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID
	cmdArgs := update.Message.CommandArguments()

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			msg := newMessage(chatID, welcomeMessage())
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send start message",
					zap.Error(err),
				)
			}

		case "next":
			_ = h.withErrorHandling(h.handleNext(from.ID, messageID))(ctx, chatID)

		case "today":
			_ = h.withErrorHandling(h.handleToday(from.ID, messageID))(ctx, chatID)

		case "random":
			_ = h.withErrorHandling(h.handleRandom(from.ID, messageID))(ctx, chatID)

		case "all":
			_ = h.withErrorHandling(h.handleAll(messageID))(ctx, chatID)

		case "range":
			_ = h.withErrorHandling(h.handleRange(cmdArgs, messageID))(ctx, chatID)

		case "progress":
			_ = h.withErrorHandling(h.handleProgress(from.ID, messageID))(ctx, chatID)

		case "quiz":
			_ = h.withErrorHandling(h.handleQuiz(from.ID, update.Message.MessageID))(ctx, chatID)

		case "settings":
			_ = h.withErrorHandling(h.handleSettings(from.ID, messageID))(ctx, chatID)

		case "help":
			msg := newMessage(chatID, helpMessage())
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send help message",
					zap.Error(err),
				)
			}

		default:
			msg := newPlainMessage(chatID, msgUnknownCommand)
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send unknown command message",
					zap.Error(err),
				)
			}
		}

		return
	}

	_ = h.withErrorHandling(h.handleNumber(update.Message.Text))(ctx, chatID)
}

// send sends a Telegram message and ignores "message is not modified" errors.
func (h *Handler) send(c tgbotapi.Chattable) error {
	_, err := h.bot.Send(c)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return err
	}
	return nil
}

// sendQuizResults sends quiz results with a keyboard.
func (h *Handler) sendQuizResults(chatID int64, session *entities.QuizSession) error {
	resultText := formatQuizResult(session)
	keyboard := buildQuizResultKeyboard()

	msg := newMessage(chatID, resultText)
	msg.ReplyMarkup = keyboard

	_, err := h.bot.Send(msg)
	return err
}

// sendQuizQuestionFromDB sends a quiz question from database with answer buttons.
func (h *Handler) sendQuizQuestionFromDB(
	chatID int64,
	session *entities.QuizSession,
	question *entities.QuizQuestion,
	name *entities.Name,
	currentNum int,
) error {
	// Build question text
	questionText := buildQuizQuestionText(question, name, currentNum, session.TotalQuestions)

	// Build keyboard with options
	keyboard := buildQuizAnswerKeyboard(session.ID, currentNum, question.Options)

	msg := newMessage(chatID, questionText)
	msg.ReplyMarkup = keyboard

	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		return err
	}

	h.quizStorage.StoreMessageID(session.ID, sentMsg.MessageID)

	return nil
}

// SendReminder sends a reminder notification to user
func (h *Handler) SendReminder(chatID int64, payload entities.ReminderPayload) error {
	text := buildReminderNotification(payload)
	keyboard := buildReminderKeyboard()

	msg := newMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	return h.send(msg)
}
