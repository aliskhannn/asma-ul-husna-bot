package telegram

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type Handler struct {
	bot             *tgbotapi.BotAPI
	logger          *zap.Logger
	nameService     NameService
	userService     UserService
	progressService ProgressService
	settingsService SettingsService
	quizService     QuizService
	quizStorage     QuizStorage
	reminderService ReminderService
}

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
) *Handler {
	return &Handler{
		bot:             bot,
		logger:          logger,
		nameService:     nameService,
		userService:     userService,
		progressService: progressService,
		settingsService: settingsService,
		quizService:     quizService,
		quizStorage:     quizStorage,
		reminderService: reminderService,
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
		update.Message.Chat.ID,
	)
	if err != nil {
		h.logger.Error("failed to ensure user",
			zap.Int64("user_id", from.ID),
			zap.Error(err),
		)
	}

	chatID := update.Message.Chat.ID
	msg := newMessage(chatID, "")

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			msg.Text = WelcomeMarkdownV2()
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send start message",
					zap.Error(err),
				)
			}

		case "random":
			_ = h.withErrorHandling(h.handleRandom(from.ID))(ctx, chatID)

		case "all":
			_ = h.withErrorHandling(func(ctx context.Context, chatID int64) error {
				return h.handleAll(ctx, chatID)
			})(ctx, chatID)

		case "range":
			_ = h.withErrorHandling(h.handleRange(update.Message.CommandArguments()))(ctx, chatID)

		case "progress":
			_ = h.withErrorHandling(h.handleProgress(from.ID))(ctx, chatID)

		case "quiz":
			_ = h.withErrorHandling(h.handleQuiz(from.ID))(ctx, chatID)

		case "settings":
			_ = h.withErrorHandling(h.handleSettings(from.ID))(ctx, chatID)

		default:
			msg = newPlainMessage(chatID, msgUnknownCommand)
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send unknown command message",
					zap.Error(err),
				)
			}
		}

		return
	}

	_ = h.withErrorHandling(h.handleNumber(update.Message.Text, from.ID))(ctx, chatID)
}

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

func (h *Handler) getCurrentQuestion(sessionID int64, currentNum int) (*entities.Question, bool) {
	questions := h.quizStorage.Get(sessionID)
	if len(questions) == 0 {
		return nil, false
	}

	idx := currentNum - 1
	if idx < 0 || idx >= len(questions) {
		return nil, false
	}

	return &questions[idx], true
}

func (h *Handler) sendQuizQuestion(
	chatID int64,
	session *entities.QuizSession,
	question *entities.Question,
	currentNum int,
) error {
	questionText := formatQuizQuestion(question, currentNum, session.TotalQuestions)
	keyboard := buildQuizAnswerKeyboard(question, session.ID, currentNum)

	msg := newMessage(chatID, questionText)
	msg.ReplyMarkup = keyboard

	return h.send(msg)
}

func (h *Handler) sendQuizResults(chatID int64, session *entities.QuizSession) error {
	resultText := formatQuizResult(session)
	keyboard := buildQuizResultKeyboard()

	msg := newMessage(chatID, resultText)
	msg.ReplyMarkup = keyboard

	_, err := h.bot.Send(msg)
	return err
}

func (h *Handler) storeQuizQuestions(sessionID int64, questions []entities.Question) {
	h.quizStorage.Store(sessionID, questions)
}

func (h *Handler) getQuizQuestions(sessionID int64) []entities.Question {
	return h.quizStorage.Get(sessionID)
}

// SendReminder sends a reminder notification to user
func (h *Handler) SendReminder(chatID int64, payload entities.ReminderPayload) error {
	text := buildReminderNotification(payload)
	keyboard := buildReminderKeyboard()

	msg := newMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	return h.send(msg)
}
