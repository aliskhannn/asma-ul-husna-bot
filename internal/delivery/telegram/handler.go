// Package telegram provides handlers for Telegram bot updates.

package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type tzWaitState struct {
	Flow            string // "onboarding" | "settings"
	ChatID          int64
	OwnerMessageID  int
	PromptMessageID int
}

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
	resetService     ResetService

	tzInputWait map[int64]tzWaitState
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
	resetService ResetService,
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
		resetService:     resetService,

		tzInputWait: make(map[int64]tzWaitState),
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

	chatID := update.Message.Chat.ID

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			_ = h.withErrorHandling(h.handleStart(from.ID))(ctx, chatID)

		case "today":
			_ = h.withErrorHandling(h.handleToday(from.ID))(ctx, chatID)

		case "random":
			_ = h.withErrorHandling(h.handleRandom(from.ID))(ctx, chatID)

		case "all":
			_ = h.withErrorHandling(h.handleAll())(ctx, chatID)

		case "progress":
			_ = h.withErrorHandling(h.handleProgress(from.ID))(ctx, chatID)

		case "quiz":
			_ = h.withErrorHandling(h.handleQuiz(from.ID))(ctx, chatID)

		case "settings":
			_ = h.withErrorHandling(h.handleSettings(from.ID))(ctx, chatID)

		case "help":
			msg := newMessage(chatID, helpMessage())
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send help message",
					zap.Error(err),
				)
			}

		case "reset":
			_ = h.withErrorHandling(h.handleReset())(ctx, chatID)

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

	text := strings.TrimSpace(update.Message.Text)

	if _, ok := h.tzInputWait[from.ID]; ok {
		_ = h.withErrorHandling(h.handleTimezoneText(text, from.ID, update.Message.MessageID))(ctx, chatID)
		return
	}

	fields := strings.Fields(text)
	if len(fields) == 2 {
		from, err1 := strconv.Atoi(fields[0])
		to, err2 := strconv.Atoi(fields[1])
		if err1 == nil && err2 == nil {
			_ = h.withErrorHandling(h.handleRangeNumbers(from, to))(ctx, chatID)
			return
		}
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
	isFirstQuiz bool,
) error {
	if isFirstQuiz && currentNum == 1 {
		if err := h.send(newMessage(chatID, buildFirstQuizMessage())); err != nil {
			return err
		}
	}

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

func (h *Handler) sendNameCard(ctx context.Context, chatID int64, nameNumber int, audioEnabled bool) error {
	msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (*entities.Name, error) {
		return h.nameService.GetByNumber(ctx, nameNumber)
	}, chatID)
	if err != nil {
		return err
	}

	if !audioEnabled {
		audio = nil
	}

	if audio != nil {
		_ = h.send(*audio)
	}
	if err := h.send(msg); err != nil {
		return err
	}
	return nil
}

// sendTodayList sends a formatted list of today's names with their learning status.
func (h *Handler) sendTodayList(ctx context.Context, chatID int64, userID int64, settings *entities.UserSettings, todayNames []int) error {
	namesPerDay := settings.NamesPerDay
	if namesPerDay <= 0 {
		namesPerDay = 1
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 *Сегодня изучаете \\(%d/%d\\):*\n\n",
		len(todayNames), namesPerDay))

	learnedCount := 0
	for i, nameNumber := range todayNames {
		name, err := h.nameService.GetByNumber(ctx, nameNumber)
		if err != nil {
			h.logger.Warn("failed to get name by number",
				zap.Error(err),
				zap.Int("name_number", nameNumber))
			continue
		}

		// Check if learned
		streak, err := h.progressService.GetStreak(ctx, userID, nameNumber)
		if err == nil && streak >= 7 {
			learnedCount++
			sb.WriteString(fmt.Sprintf("✅ %d\\. %s\n", i+1, bold(name.Translation)))
		} else {
			sb.WriteString(fmt.Sprintf("⏳ %d\\. %s\n", i+1, bold(name.Translation)))
		}
	}

	sb.WriteString("\n")

	if learnedCount == len(todayNames) {
		sb.WriteString("✅ *Все имена изучены\\!*\n\n")
		if len(todayNames) < namesPerDay {
			sb.WriteString(fmt.Sprintf("Можете добавить ещё %d имя\\(ён\\) через /next\\.",
				namesPerDay-len(todayNames)))
		}
	} else {
		sb.WriteString(fmt.Sprintf("⏳ Прогресс: %d/%d изучено\n\n",
			learnedCount, len(todayNames)))
		sb.WriteString("Используйте /next чтобы продолжить\\.")
	}

	msg := newMessage(chatID, sb.String())
	return h.send(msg)
}

// SendReminder sends a reminder notification to user
func (h *Handler) SendReminder(chatID int64, payload entities.ReminderPayload) error {
	text := buildReminderNotification(payload)
	keyboard := buildReminderKeyboard()

	msg := newMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	return h.send(msg)
}

func (h *Handler) removeInlineKeyboard(chatID int64, messageID int) {
	edit := tgbotapi.NewEditMessageReplyMarkup(
		chatID, messageID,
		tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}},
	)
	_, _ = h.bot.Request(edit)
}

func (h *Handler) setTZWaitState(userID int64, st tzWaitState) {
	if old, ok := h.tzInputWait[userID]; ok && old.PromptMessageID != 0 {
		_ = h.send(tgbotapi.NewDeleteMessage(old.ChatID, old.PromptMessageID))
	}
	h.tzInputWait[userID] = st
}
