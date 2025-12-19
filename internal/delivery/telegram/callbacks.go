package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

// handleCallback routes callback queries to appropriate handlers.
func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	data := decodeCallback(cb.Data)

	switch data.Action {
	case actionName:
		h.withCallbackErrorHandling(h.handleNameCallback)(ctx, cb)
	case actionRange:
		h.withCallbackErrorHandling(h.handleRangeCallback)(ctx, cb)
	case actionSettings:
		h.withCallbackErrorHandling(h.handleSettingsCallback)(ctx, cb)
	case actionQuiz:
		h.withCallbackErrorHandling(h.handleQuizCallback)(ctx, cb)
	case actionProgress:
		h.withCallbackErrorHandling(h.handleProgressCallback)(ctx, cb)
	default:
		h.logger.Warn("unknown callback action",
			zap.String("action", data.Action),
			zap.String("raw", data.Raw),
		)
	}

	// Remove the user's "loading clock".
	answer := tgbotapi.NewCallback(cb.ID, "")
	if _, err := h.bot.Request(answer); err != nil {
		h.logger.Error("callback answer error",
			zap.Error(err),
			zap.String("data", cb.Data),
		)
	}
}

func (h *Handler) handleNameCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	if cb.Message == nil {
		return nil
	}

	data := decodeCallback(cb.Data)
	if len(data.Params) != 1 {
		h.logger.Warn("invalid name callback params", zap.String("raw", data.Raw))
		return nil
	}

	page, err := strconv.Atoi(data.Params[0])
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
		msg := newPlainMessage(cb.Message.Chat.ID, msgNameUnavailable)
		return h.send(msg)
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

	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, text)
	if kb != nil {
		edit.ReplyMarkup = kb
	}

	return h.send(edit)
}

// handleRangeCallback handles pagination for range-based name view.
func (h *Handler) handleRangeCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	if cb.Message == nil {
		return nil
	}

	data := decodeCallback(cb.Data)
	if len(data.Params) != 3 {
		h.logger.Warn("invalid range callback params", zap.String("raw", data.Raw))
		return nil
	}

	page, err1 := strconv.Atoi(data.Params[0])
	from, err2 := strconv.Atoi(data.Params[1])
	to, err3 := strconv.Atoi(data.Params[2])

	if err1 != nil || err2 != nil || err3 != nil || page < 0 || from < 1 || to > 99 || from > to {
		h.logger.Warn("invalid range callback values",
			zap.String("data", cb.Data),
			zap.Errors("errors", []error{err1, err2, err3}),
		)
		return nil
	}

	names, err := h.getAllNames(ctx)
	if err != nil {
		return err
	}

	if names == nil {
		msg := newPlainMessage(cb.Message.Chat.ID, msgNameUnavailable)
		return h.send(msg)
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
	prevData := buildRangeCallback(page-1, from, to)
	nextData := buildRangeCallback(page+1, from, to)
	kb := buildNameKeyboard(page, totalPages, prevData, nextData)

	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, text)
	if kb != nil {
		edit.ReplyMarkup = kb
	}

	return h.send(edit)
}

// handleSettingsCallback handles all settings-related callbacks.
func (h *Handler) handleSettingsCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	if cb.Message == nil {
		return nil
	}

	data := decodeCallback(cb.Data)
	if len(data.Params) < 1 {
		h.logger.Warn("invalid settings callback", zap.String("raw", data.Raw))
		return nil
	}

	subAction := data.Params[0]

	// Menu navigation (no value).
	if len(data.Params) == 1 {
		return h.handleSettingsNavigation(ctx, cb, subAction)
	}

	// Apply setting value.
	value := data.Params[1]
	return h.applySettingValue(ctx, cb, subAction, value)
}

// handleSettingsNavigation shows settings menus.
func (h *Handler) handleSettingsNavigation(ctx context.Context, cb *tgbotapi.CallbackQuery, subAction string) error {
	switch subAction {
	case settingsMenu:
		return h.showSettingsMenu(ctx, cb)
	case settingsNamesPerDay:
		msg := "📚 " + bold("Сколько новых имён изучать в день?") + "\n\n" +
			md("Выберите интенсивность обучения:")
		return h.showSettingsSubmenu(cb, msg, buildNamesPerDayKeyboard())

	case settingsQuizMode:
		msg := "🎲 " + bold("Режим квиза") + "\n\n" +
			md("Выберите, какие имена включать в квиз: только новые, только на повторение или оба варианта.")
		return h.showSettingsSubmenu(cb, msg, buildQuizModeKeyboard())

	default:
		h.logger.Warn("unknown settings sub-action", zap.String("sub_action", subAction))
		return nil
	}
}

// applySettingValue applies a new setting value.
func (h *Handler) applySettingValue(ctx context.Context, cb *tgbotapi.CallbackQuery, subAction, value string) error {
	switch subAction {
	case settingsNamesPerDay:
		return h.applyNamesPerDay(ctx, cb, value)
	case settingsQuizMode:
		return h.applyQuizMode(ctx, cb, value)
	default:
		h.logger.Warn("unknown settings sub-action with value", zap.String("sub_action", subAction))
		return nil
	}
}

// showSettingsMenu displays the main settings menu.
func (h *Handler) showSettingsMenu(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	text, keyboard, err := h.RenderSettings(ctx, cb.From.ID)
	if err != nil {
		msg := newPlainMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
		return h.send(msg)
	}

	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ReplyMarkup = &keyboard
	return h.send(edit)
}

// showSettingsSubmenu displays a settings submenu.
func (h *Handler) showSettingsSubmenu(cb *tgbotapi.CallbackQuery, message string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, message)
	edit.ReplyMarkup = &keyboard
	return h.send(edit)
}

// applyNamesPerDay updates names per day setting.
func (h *Handler) applyNamesPerDay(ctx context.Context, cb *tgbotapi.CallbackQuery, value string) error {
	v, err := strconv.Atoi(value)
	if err != nil || v < 1 || v > 20 {
		h.logger.Warn("invalid names_per_day value",
			zap.String("value", value),
			zap.Error(err),
		)
		return nil
	}

	if err := h.settingsService.UpdateNamesPerDay(ctx, cb.From.ID, v); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newPlainMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}
		return err
	}

	return h.confirmSettingAndShowMenu(ctx, cb, fmt.Sprintf("Имён в день: %d", v))
}

// applyQuizMode updates quiz mode setting.
func (h *Handler) applyQuizMode(ctx context.Context, cb *tgbotapi.CallbackQuery, value string) error {
	if err := h.settingsService.UpdateQuizMode(ctx, cb.From.ID, value); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newPlainMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}
		return err
	}

	return h.confirmSettingAndShowMenu(ctx, cb, fmt.Sprintf("Режим квиза: %s", formatQuizMode(value)))
}

// confirmSettingAndShowMenu shows confirmation and returns to settings menu.
func (h *Handler) confirmSettingAndShowMenu(ctx context.Context, cb *tgbotapi.CallbackQuery, confirmText string) error {
	confirm := tgbotapi.NewCallback(cb.ID, confirmText)
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send confirmation", zap.Error(err))
	}
	return h.showSettingsMenu(ctx, cb)
}

// handleQuizCallback handles quiz-related callbacks.
func (h *Handler) handleQuizCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	data := decodeCallback(cb.Data)

	if len(data.Params) < 1 {
		return fmt.Errorf("invalid quiz callback: no params")
	}

	// Check if it's quiz start.
	if data.Params[0] == quizStart {
		return h.handleQuizStart(ctx, cb)
	}

	// Otherwise, it's a quiz answer: quiz:{sessionID}:{questionNum}:{answerIndex}
	if len(data.Params) != 3 {
		return fmt.Errorf("invalid quiz answer callback: expected 3 params, got %d", len(data.Params))
	}

	return h.handleQuizAnswer(ctx, cb, data)
}

// handleQuizStart starts a new quiz session.
func (h *Handler) handleQuizStart(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	chatID := cb.Message.Chat.ID
	userID := cb.From.ID

	// Delete previous message.
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID)
	_, _ = h.bot.Send(deleteMsg)

	settings, err := h.settingsService.GetOrCreate(ctx, userID)
	if err != nil {
		msg := newPlainMessage(chatID, msgSettingsUnavailable)
		return h.send(msg)
	}

	mode := settings.QuizMode
	session, questions, err := h.quizService.GenerateQuiz(ctx, userID, mode)
	if err != nil || len(questions) == 0 {
		msg := newPlainMessage(chatID, msgQuizUnavailable)
		return h.send(msg)
	}

	h.storeQuizQuestions(session.ID, questions)

	startMsg := newMessage(chatID, buildQuizStartMessage(mode))
	if err := h.send(startMsg); err != nil {
		return err
	}

	if err := h.sendQuizQuestion(chatID, session, &questions[0], 1); err != nil {
		return err
	}

	return h.answerCallback(cb.ID, "")
}

// handleQuizAnswer processes a quiz answer.
func (h *Handler) handleQuizAnswer(ctx context.Context, cb *tgbotapi.CallbackQuery, data callbackData) error {
	sessionID, err := strconv.ParseInt(data.Params[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	questionNum, err := strconv.Atoi(data.Params[1])
	if err != nil {
		return fmt.Errorf("invalid question number: %w", err)
	}

	answerIndex, err := strconv.Atoi(data.Params[2])
	if err != nil {
		return fmt.Errorf("invalid answer index: %w", err)
	}

	userID := cb.From.ID
	chatID := cb.Message.Chat.ID

	session, err := h.quizService.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if questionNum != session.CurrentQuestionNum {
		return h.answerCallback(cb.ID, "Вопрос уже был отвечен")
	}

	questions := h.getQuizQuestions(sessionID)
	if questions == nil || questionNum > len(questions) {
		return h.answerCallback(cb.ID, "Вопрос не найден")
	}

	question := &questions[questionNum-1]
	isCorrect := answerIndex == question.CorrectIndex

	// Record SRS review.
	quality := entities.QualityGood
	if !isCorrect {
		quality = entities.QualityFail
	}

	if err := h.progressService.RecordReviewSRS(ctx, userID, question.NameNumber, quality); err != nil {
		h.logger.Error("failed to record SRS review", zap.Error(err))
		// Continue execution, not a critical error.
	}

	// Save answer.
	answer, err := h.quizService.CheckAndSaveAnswer(ctx, userID, session, question, answerIndex)
	if err != nil {
		h.logger.Error("failed to check answer", zap.Error(err))
		return h.answerCallback(cb.ID, "Ошибка при проверке ответа")
	}

	// Delete question message.
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID)
	_, _ = h.bot.Send(deleteMsg)

	// Send feedback.
	feedbackText := formatAnswerFeedback(answer.IsCorrect, question.CorrectAnswer)
	feedbackMsg := newMessage(chatID, feedbackText)
	_, err = h.bot.Send(feedbackMsg)
	if err != nil {
		h.logger.Error("failed to send feedback", zap.Error(err))
	}

	// Check if quiz is completed.
	if session.SessionStatus == "completed" {
		return h.sendQuizResults(chatID, session)
	}

	// Send next question.
	if session.CurrentQuestionNum <= len(questions) {
		nextQuestion := &questions[session.CurrentQuestionNum-1]
		err = h.sendQuizQuestion(chatID, session, nextQuestion, session.CurrentQuestionNum)
		if err != nil {
			h.logger.Error("failed to send next question", zap.Error(err))
		}
	}

	return h.answerCallback(cb.ID, "")
}

// handleProgressCallback shows user progress.
func (h *Handler) handleProgressCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	if cb.Message == nil {
		return nil
	}

	text, keyboard, err := h.RenderProgress(ctx, cb.From.ID, true)
	if err != nil {
		msg := newPlainMessage(cb.Message.Chat.ID, msgProgressUnavailable)
		return h.send(msg)
	}

	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, text)
	if keyboard != nil {
		edit.ReplyMarkup = keyboard
	}

	return h.send(edit)
}

// answerCallback sends a callback answer (removes loading indicator).
func (h *Handler) answerCallback(callbackID, text string) error {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := h.bot.Request(callback)
	return err
}
