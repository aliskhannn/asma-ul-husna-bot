// Package telegram provides handlers for Telegram bot updates.

package telegram

import (
	"context"
	"fmt"
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
	resetService     ResetService
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
	cmdArgs := update.Message.CommandArguments()

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			_ = h.withErrorHandling(h.handleStart(from.ID))(ctx, chatID)

		case "next":
			_ = h.withErrorHandling(h.handleNext(from.ID))(ctx, chatID)

		case "today":
			_ = h.withErrorHandling(h.handleToday(from.ID))(ctx, chatID)

		case "random":
			_ = h.withErrorHandling(h.handleRandom(from.ID))(ctx, chatID)

		case "all":
			_ = h.withErrorHandling(h.handleAll())(ctx, chatID)

		case "range":
			_ = h.withErrorHandling(h.handleRange(cmdArgs))(ctx, chatID)

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
	ctx context.Context,
	userID int64,
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

func (h *Handler) sendNameCard(ctx context.Context, chatID int64, messageID int, nameNumber int) error {
	return h.sendNameCardWithPrefix(ctx, chatID, messageID, "", nameNumber, "", "")
}

func (h *Handler) sendNameCardWithPrefix(ctx context.Context, chatID int64, messageID int, cmd string, nameNumber int, prefix, suffix string) error {
	msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (*entities.Name, error) {
		return h.nameService.GetByNumber(ctx, nameNumber)
	}, chatID, prefix, suffix)
	if err != nil {
		return err
	}

	if cmd == "next" {
		msg.ReplyMarkup = nextCardKeyboard()
	}

	_, _ = h.bot.Send(tgbotapi.NewDeleteMessage(chatID, messageID))

	if audio != nil {
		_ = h.send(*audio)
	}
	if err := h.send(msg); err != nil {
		return err
	}
	return nil
}

func (h *Handler) sendNextLimitReached(chatID int64, introducedToday, namesPerDay int) error {
	var sb strings.Builder

	sb.WriteString(md("✅ Вы достигли дневного лимита ("))
	sb.WriteString(bold(fmt.Sprintf("%d/%d", introducedToday, namesPerDay)))
	sb.WriteString(md(")\n\n"))

	sb.WriteString(md("Что можно сделать дальше:\n"))
	sb.WriteString(md("• Нажмите "))
	sb.WriteString(bold("«🧠 Квиз»"))
	sb.WriteString(md(" — закрепить сегодняшние имена\n"))

	sb.WriteString(md("• Нажмите "))
	sb.WriteString(bold("«📅 Сегодня»"))
	sb.WriteString(md(" — посмотреть план\n"))

	sb.WriteString(md("• Нажмите "))
	sb.WriteString(bold("«⚙️ Настройки»"))
	sb.WriteString(md(" — увеличить лимит «Имён в день»\n\n"))

	sb.WriteString(md("Вернитесь завтра за новыми именами!"))

	msg := newMessage(chatID, sb.String())
	kb := nextCardKeyboard()
	msg.ReplyMarkup = kb

	return h.send(msg)
}

func (h *Handler) sendNextBlockedNeedQuiz(chatID int64, namesPerDay int) error {
	text := fmt.Sprintf(
		"📚 Сегодня вы уже изучаете %d %s.\n\n"+
			"Нажмите кнопку «🧠 Квиз»: нужно 2 правильных ответа, чтобы открыть следующее имя.\n\n"+
			"💡 Или откройте «⚙️ Настройки» и увеличьте лимит «Имён в день».",
		namesPerDay, formatNamesCount(namesPerDay),
	)

	msg := newMessage(chatID, md(text))
	kb := nextCardKeyboard()
	msg.ReplyMarkup = kb

	return h.send(msg)
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
