package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	case actionReminder:
		h.withCallbackErrorHandling(h.handleReminderCallback)(ctx, cb)
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

// handleNameCallback handles pagination for names list.
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

	if len(data.Params) == 1 {
		return h.handleSettingsNavigation(ctx, cb, subAction)
	}

	value := data.Params[1]

	if subAction == settingsReminders {
		return h.applyReminderSetting(ctx, cb, value, data.Params)
	}

	return h.applySettingValue(ctx, cb, subAction, value)
}

// handleSettingsNavigation shows settings menus.
func (h *Handler) handleSettingsNavigation(ctx context.Context, cb *tgbotapi.CallbackQuery, subAction string) error {
	switch subAction {
	case settingsMenu:
		return h.showSettingsMenu(ctx, cb)

	case settingsLearningMode:
		msg := "🎯 " + bold("Режим обучения") + "\n\n" + learningModeDescription()
		return h.showSettingsSubmenu(cb, msg, buildLearningModeKeyboard())

	case settingsNamesPerDay:
		msg := "📚 " + bold("Сколько новых имён изучать в день?") + "\n\n" +
			md("Выберите интенсивность обучения:")
		return h.showSettingsSubmenu(cb, msg, buildNamesPerDayKeyboard())

	case settingsQuizMode:
		msg := "🎲 " + bold("Режим квиза") + "\n\n" +
			md("Выберите, какие имена включать в квиз: только новые, только на повторение или оба варианта.")
		return h.showSettingsSubmenu(cb, msg, buildQuizModeKeyboard())

	case settingsReminders:
		return h.showReminderSettings(ctx, cb)

	default:
		h.logger.Warn("unknown settings sub-action", zap.String("sub_action", subAction))
		return nil
	}
}

// applySettingValue applies a new setting value.
func (h *Handler) applySettingValue(ctx context.Context, cb *tgbotapi.CallbackQuery, subAction, value string) error {
	switch subAction {
	case settingsLearningMode:
		return h.applyLearningMode(ctx, cb, value)
	case settingsNamesPerDay:
		return h.applyNamesPerDay(ctx, cb, value)
	case settingsQuizMode:
		return h.applyQuizMode(ctx, cb, value)
	default:
		h.logger.Warn("unknown settings sub-action with value", zap.String("sub_action", subAction))
		return nil
	}
}

func (h *Handler) applyLearningMode(ctx context.Context, cb *tgbotapi.CallbackQuery, value string) error {
	if value != "guided" && value != "free" {
		h.logger.Warn("invalid learning_mode value", zap.String("value", value))
		return nil
	}

	if err := h.settingsService.UpdateLearningMode(ctx, cb.From.ID, value); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newPlainMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}
		return err
	}

	modeText := formatLearningMode(entities.LearningMode(value))
	return h.confirmSettingAndShowMenu(ctx, cb, fmt.Sprintf("Режим обучения: %s", modeText))
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

// showReminderSettings displays reminder settings screen
func (h *Handler) showReminderSettings(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	reminder, err := h.reminderService.GetByUserID(ctx, cb.From.ID)
	if err != nil {
		msg := newPlainMessage(cb.Message.Chat.ID, msgInternalError)
		return h.send(msg)
	}

	text := buildReminderSettingsMessage(reminder)
	keyboard := buildRemindersKeyboard(reminder)

	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, text)
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

// handleReminderCallback handles reminder action callbacks
func (h *Handler) handleReminderCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	data := decodeCallback(cb.Data)

	if len(data.Params) == 0 {
		return fmt.Errorf("missing reminder action")
	}

	action := data.Params[0]
	userID := cb.From.ID
	chatID := cb.Message.Chat.ID

	switch action {
	case reminderStartQuiz:
		answer := tgbotapi.NewCallback(cb.ID, "Запускаю квиз...")
		if _, err := h.bot.Request(answer); err != nil {
			h.logger.Error("failed to answer callback", zap.Error(err))
		}

		deleteMsg := tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID)
		if _, err := h.bot.Request(deleteMsg); err != nil {
			h.logger.Error("failed to delete message", zap.Error(err))
		}

		return h.handleQuiz(userID, cb.Message.MessageID)(ctx, chatID)

	case reminderSnooze:
		if err := h.reminderService.SnoozeReminder(ctx, userID, time.Hour); err != nil {
			return err
		}

		answer := tgbotapi.NewCallback(cb.ID, "⏰ Напомню через 1 час")
		if _, err := h.bot.Request(answer); err != nil {
			h.logger.Error("failed to answer callback", zap.Error(err))
		}

		deleteMsg := tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID)
		if _, err := h.bot.Request(deleteMsg); err != nil {
			h.logger.Error("failed to delete message", zap.Error(err))
		}

		return nil

	case reminderDisable:
		if err := h.reminderService.DisableReminder(ctx, userID); err != nil {
			return err
		}

		answer := tgbotapi.NewCallback(cb.ID, "🔕 Напоминания выключены")
		if _, err := h.bot.Request(answer); err != nil {
			h.logger.Error("failed to answer callback", zap.Error(err))
		}

		deleteMsg := tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID)
		if _, err := h.bot.Request(deleteMsg); err != nil {
			h.logger.Error("failed to delete message", zap.Error(err))
		}

		return nil

	default:
		return fmt.Errorf("unknown reminder action: %s", action)
	}
}

// applyReminderSetting applies reminder setting changes
func (h *Handler) applyReminderSetting(ctx context.Context, cb *tgbotapi.CallbackQuery, value string, params []string) error {
	userID := cb.From.ID

	switch value {
	case reminderToggle:
		if err := h.reminderService.ToggleReminder(ctx, userID); err != nil {
			msg := newPlainMessage(cb.Message.Chat.ID, msgInternalError)
			return h.send(msg)
		}
		return h.showReminderSettings(ctx, cb)

	case "frequency":
		return h.showFrequencyMenu(ctx, cb)

	case "time":
		if len(params) < 4 {
			return h.showTimeWindowMenu(ctx, cb)
		}

		startTime := strings.ReplaceAll(params[2], "-", ":")
		endTime := strings.ReplaceAll(params[3], "-", ":")

		if err := h.reminderService.SetReminderTimeWindow(ctx, userID, startTime, endTime); err != nil {
			msg := newPlainMessage(cb.Message.Chat.ID, msgInternalError)
			return h.send(msg)
		}

		confirmText := fmt.Sprintf("⏰ Время: %s - %s", startTime[:5], endTime[:5])
		return h.confirmSettingAndShowReminderSettings(ctx, cb, confirmText)

	case "freq":
		if len(params) < 3 {
			h.logger.Warn("invalid frequency params", zap.Strings("params", params))
			return nil
		}

		interval, err := formatIntervalHoursString(params[2])
		if err != nil {
			msg := newPlainMessage(cb.Message.Chat.ID, msgInvalidIntervalHours)
			return h.send(msg)
		}

		if err := h.reminderService.SetReminderIntervalHours(ctx, userID, interval); err != nil {
			msg := newPlainMessage(cb.Message.Chat.ID, msgInternalError)
			return h.send(msg)
		}

		confirmText := fmt.Sprintf("📅 Частота: %s", formatIntervalHoursInt(interval))
		return h.confirmSettingAndShowReminderSettings(ctx, cb, confirmText)

	default:
		h.logger.Warn("unknown reminder sub-action", zap.String("value", value), zap.Strings("params", params))
		return nil
	}
}

// showFrequencyMenu displays frequency selection menu
func (h *Handler) showFrequencyMenu(_ context.Context, cb *tgbotapi.CallbackQuery) error {
	text := "📅 " + bold("Как часто отправлять напоминания?") + "\n\n" +
		md("Выберите частоту напоминаний в день:")

	keyboard := buildFrequencyKeyboard()

	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ReplyMarkup = &keyboard
	return h.send(edit)
}

// showTimeWindowMenu displays time window selection menu
func (h *Handler) showTimeWindowMenu(_ context.Context, cb *tgbotapi.CallbackQuery) error {
	text := "⏰ " + bold("В какое время отправлять напоминания?") + "\n\n" +
		md("Выберите временной диапазон для напоминаний:")

	keyboard := buildTimeWindowKeyboard()

	edit := newEdit(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ReplyMarkup = &keyboard
	return h.send(edit)
}

// confirmSettingAndShowMenu shows confirmation and returns to settings menu.
func (h *Handler) confirmSettingAndShowMenu(ctx context.Context, cb *tgbotapi.CallbackQuery, confirmText string) error {
	confirm := tgbotapi.NewCallback(cb.ID, confirmText)
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send confirmation", zap.Error(err))
	}
	return h.showSettingsMenu(ctx, cb)
}

// confirmSettingAndShowReminderSettings shows confirmation and returns to reminder settings
func (h *Handler) confirmSettingAndShowReminderSettings(ctx context.Context, cb *tgbotapi.CallbackQuery, confirmText string) error {
	confirm := tgbotapi.NewCallback(cb.ID, confirmText)
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send confirmation", zap.Error(err))
	}

	return h.showReminderSettings(ctx, cb)
}

// handleQuizCallback handles quiz-related callbacks.
func (h *Handler) handleQuizCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	data := decodeCallback(cb.Data)

	// Handle "start quiz" action
	if len(data.Params) == 1 && data.Params[0] == quizStart {
		return h.handleQuiz(cb.From.ID, cb.Message.MessageID)(ctx, cb.Message.Chat.ID)
	}

	// Handle quiz answer: quiz:sessionID:questionNum:answerIndex
	if len(data.Params) < 3 {
		h.logger.Warn("invalid quiz callback params", zap.String("raw", data.Raw))
		return nil
	}

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

	// Submit answer with index
	result, err := h.quizService.SubmitAnswer(ctx, sessionID, userID, strconv.Itoa(answerIndex))
	if err != nil {
		if strings.Contains(err.Error(), "already submitted") {
			return h.answerCallback(cb.ID, "Ответ уже отправлен")
		}
		h.logger.Error("failed to submit answer",
			zap.Error(err),
			zap.Int64("session_id", sessionID),
			zap.Int("question_num", questionNum),
			zap.Int("answer_index", answerIndex),
		)
		return h.answerCallback(cb.ID, "Ошибка при проверке ответа")
	}

	// Delete question message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID)
	_ = h.send(deleteMsg)

	// Send feedback
	feedbackText := formatAnswerFeedback(result.IsCorrect, result.CorrectAnswer)
	feedbackMsg := newMessage(chatID, feedbackText)
	_, err = h.bot.Send(feedbackMsg)
	if err != nil {
		h.logger.Error("failed to send feedback", zap.Error(err))
	}

	// Check if quiz is completed
	if result.IsSessionComplete {
		// Clear storage
		h.quizStorage.Delete(sessionID)

		// Build session summary
		completedSession := &entities.QuizSession{
			ID:             sessionID,
			CorrectAnswers: result.Score,
			TotalQuestions: result.Total,
			SessionStatus:  "completed",
		}
		return h.sendQuizResults(chatID, completedSession)
	}

	// Send next question
	nextQuestionNum := questionNum + 1
	question, nextName, err := h.quizService.GetCurrentQuestion(ctx, sessionID, nextQuestionNum)
	if err != nil {
		h.logger.Error("failed to get next question",
			zap.Error(err),
			zap.Int64("session_id", sessionID),
			zap.Int("next_question_num", nextQuestionNum),
		)
		return h.answerCallback(cb.ID, "Ошибка при загрузке следующего вопроса")
	}

	// Get active session to pass correct data
	session, err := h.quizService.GetActiveSession(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get active session",
			zap.Error(err),
			zap.Int64("user_id", userID),
		)
		return nil
	}

	err = h.sendQuizQuestionFromDB(chatID, session, question, nextName, nextQuestionNum)
	if err != nil {
		h.logger.Error("failed to send next question", zap.Error(err))
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
