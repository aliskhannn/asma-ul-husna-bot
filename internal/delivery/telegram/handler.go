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
	RecordReviewSRS(ctx context.Context, userID int64, nameNumber int, quality entities.AnswerQuality) error
}

type SettingsService interface {
	GetOrCreate(ctx context.Context, userID int64) (*entities.UserSettings, error)
	UpdateNamesPerDay(ctx context.Context, userID int64, namesPerDay int) error
	UpdateQuizLength(ctx context.Context, userID int64, quizLength int) error
	UpdateQuizMode(ctx context.Context, userID int64, quizMode string) error
	ToggleTransliteration(ctx context.Context, userID int64) error
	ToggleAudio(ctx context.Context, userID int64) error
}

type QuizService interface {
	GenerateQuiz(ctx context.Context, userID int64, mode string) (*entities.QuizSession, []entities.Question, error)
	GetSession(ctx context.Context, sessionID int64) (*entities.QuizSession, error)
	CheckAndSaveAnswer(ctx context.Context, userID int64, session *entities.QuizSession, q *entities.Question, selectedIndex int) (*entities.QuizAnswer, error)
}

type QuizStorage interface {
	Store(sessionID int64, questions []entities.Question)
	Get(sessionID int64) []entities.Question
	Delete(sessionID int64)
}

type Handler struct {
	bot             *tgbotapi.BotAPI
	logger          *zap.Logger
	nameService     NameService
	userService     UserService
	progressService ProgressService
	settingsService SettingsService
	quizService     QuizService
	quizStorage     QuizStorage
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
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send start message",
					zap.Error(err),
				)
			}

		case "random":
			_ = h.withErrorHandling(h.randomHandler(from.ID))(ctx, chatID)

		case "all":
			_ = h.withErrorHandling(func(ctx context.Context, chatID int64) error {
				return h.allCommandHandler(ctx, chatID)
			})(ctx, chatID)

		case "range":
			_ = h.withErrorHandling(h.rangeCommandHandler(update.Message.CommandArguments()))(ctx, chatID)

		case "progress":
			_ = h.withErrorHandling(h.progressHandler(from.ID))(ctx, chatID)

		case "quiz":
			_ = h.withErrorHandling(h.quizHandler(from.ID))(ctx, chatID)

		case "settings":
			_ = h.withErrorHandling(h.settingsHandler(from.ID))(ctx, chatID)

		default:
			msg.Text = msgUnknownCommand
			if err := h.send(msg); err != nil {
				h.logger.Error("failed to send unknown command message",
					zap.Error(err),
				)
			}
		}

		return
	}

	_ = h.withErrorHandling(h.numberHandler(update.Message.Text, from.ID))(ctx, chatID)
}

func (h *Handler) send(c tgbotapi.Chattable) error {
	_, err := h.bot.Send(c)
	return err
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

	msg := newHTMLMessage(chatID, questionText)
	msg.ReplyMarkup = keyboard

	return h.send(msg)
}

func (h *Handler) sendQuizResults(chatID int64, session *entities.QuizSession) error {
	resultText := formatQuizResult(session)
	keyboard := buildQuizResultKeyboard()

	msg := newHTMLMessage(chatID, resultText)
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
