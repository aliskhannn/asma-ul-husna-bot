package telegram

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

func (h *Handler) handleStart(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		isNewUser, err := h.userService.EnsureUser(ctx, userID, chatID)
		if err != nil {
			return h.send(newPlainMessage(chatID, msgInternalError))
		}

		stats, err := h.progressService.GetProgressSummary(ctx, userID)
		if err != nil {
			msg := newPlainMessage(chatID, msgInternalError)
			return h.send(msg)
		}

		msg := newMessage(chatID, welcomeMessage(isNewUser, stats))

		if isNewUser {
			kb := onboardingStep1Keyboard()
			msg.ReplyMarkup = kb
		} else {
			kb := welcomeReturningKeyboard()
			msg.ReplyMarkup = kb
		}

		return h.send(msg)
	}
}

// handleNumber processes numeric input and displays the corresponding name.
func (h *Handler) handleNumber(numStr string) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		n, err := strconv.Atoi(numStr)
		if err != nil {
			msg := newPlainMessage(chatID, msgIncorrectNameNumber)
			return h.send(msg)
		}

		if n < 1 || n > 99 {
			msg := newPlainMessage(chatID, msgOutOfRangeNumber)
			return h.send(msg)
		}

		msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (*entities.Name, error) {
			return h.nameService.GetByNumber(ctx, n)
		}, chatID)
		if err != nil {
			return err
		}

		if audio != nil {
			_ = h.send(*audio)
		}
		if err = h.send(msg); err != nil {
			return err
		}

		return nil
	}
}

func (h *Handler) handleTimezoneText(text string, userID int64, userMsgID int) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		st, ok := h.tzInputWait[userID]
		if !ok {
			return nil
		}

		tz, ok := normalizeUTCOffset(text)
		if !ok {
			msg := newPlainMessage(chatID, "Не понял формат. Пример: UTC+3 или UTC+5:30")
			msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
			return h.send(msg)
		}

		if err := h.settingsService.UpdateTimezone(ctx, userID, tz); err != nil {
			return h.send(newPlainMessage(chatID, msgInternalError))
		}

		// Cleanup messages (best-effort)
		if st.PromptMessageID != 0 {
			_ = h.send(tgbotapi.NewDeleteMessage(st.ChatID, st.PromptMessageID))
		}
		if userMsgID != 0 {
			_ = h.send(tgbotapi.NewDeleteMessage(chatID, userMsgID))
		}

		delete(h.tzInputWait, userID)

		switch st.Flow {
		case "onboarding":
			edit := newEdit(st.ChatID, st.OwnerMessageID, onboardingCompleteMessage())
			kb := onboardingCompleteKeyboard()
			edit.ReplyMarkup = &kb
			return h.send(edit)

		case "settings":
			settings, err := h.settingsService.GetOrCreate(ctx, userID)
			if err != nil {
				msg := newPlainMessage(chatID, msgInternalError)
				return h.send(msg)
			}

			// Return to reminders settings (edit the settings message, not onboarding)
			rem, err := h.reminderService.GetByUserID(ctx, userID)
			if err != nil {
				return h.send(newPlainMessage(chatID, fmt.Sprintf("🌍 Часовой пояс сохранён: %s", tz)))
			}

			edit := newEdit(st.ChatID, st.OwnerMessageID, buildReminderSettingsMessage(settings.Timezone, rem))
			kb := buildRemindersKeyboard(rem)
			edit.ReplyMarkup = &kb

			// optional: show toast via callback isn't possible here; send a short message if needed
			_ = h.send(newPlainMessage(chatID, fmt.Sprintf("🌍 Часовой пояс: %s", tz)))

			return h.send(edit)

		default:
			return nil
		}
	}
}

func normalizeUTCOffset(input string) (string, bool) {
	s := strings.TrimSpace(input)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ToUpper(s)

	if strings.HasPrefix(s, "UTC") {
		s = s[3:]
		if s == "" {
			return "UTC+0", true
		}
	}

	if !(strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-")) {
		return "", false
	}

	sign := s[:1]
	rest := s[1:]

	hh := rest
	mm := "00"
	if strings.Contains(rest, ":") {
		parts := strings.Split(rest, ":")
		if len(parts) != 2 {
			return "", false
		}
		hh, mm = parts[0], parts[1]
	}

	h, err := strconv.Atoi(hh)
	if err != nil {
		return "", false
	}
	m, err := strconv.Atoi(mm)
	if err != nil {
		return "", false
	}

	if h < 0 || h > 14 || m < 0 || m >= 60 {
		return "", false
	}
	if m%15 != 0 {
		return "", false
	}

	return fmt.Sprintf("UTC%s%d:%02d", sign, h, m), true
}

func (h *Handler) handleToday(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		return h.handleTodayPage(userID)(ctx, chatID, 0, 0)
	}
}

func (h *Handler) handleTodayPage(userID int64) func(ctx context.Context, chatID int64, messageID int, page int) error {
	return func(ctx context.Context, chatID int64, messageID int, page int) error {
		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil || settings == nil {
			settings = entities.NewUserSettings(userID)
		}
		namesPerDay := settings.NamesPerDay
		if namesPerDay <= 0 {
			namesPerDay = 1
		}

		// ensure today's plan exists (debt + new up to quota)
		err = h.dailyNameService.EnsureTodayPlan(
			ctx,
			userID,
			settings.Timezone,
			namesPerDay,
		)
		if err != nil {
			return h.send(newPlainMessage(chatID, msgInternalError))
		}

		todayNames, err := h.dailyNameService.GetTodayNamesTZ(ctx, userID, settings.Timezone)
		if err != nil {
			return h.send(newPlainMessage(chatID, msgInternalError))
		}
		if len(todayNames) == 0 {
			return h.send(newPlainMessage(chatID, "📚 На сегодня пока нет имён.\n\nНажмите /new, чтобы открыть новое имя."))
		}

		if page < 0 {
			page = 0
		}
		if page >= len(todayNames) {
			page = len(todayNames) - 1
		}

		nameNumber := todayNames[page]

		// статус (✅ mastered, ⏳ иначе)
		status := "⏳"
		pMap, _ := h.progressService.GetByNumbers(ctx, userID, []int{nameNumber})
		if p := pMap[nameNumber]; p != nil && p.Phase == entities.PhaseMastered {
			status = "✅"
		}

		prefix := md(fmt.Sprintf("📅 Сегодня: %s %d/%d\n\n", status, page+1, len(todayNames)))

		name, err := h.nameService.GetByNumber(ctx, nameNumber)
		if err != nil {
			return h.send(newPlainMessage(chatID, msgNameUnavailable))
		}

		text := prefix + buildNameCardText(name)

		kb := todayCardsKeyboard(page, len(todayNames), name.Number)

		if messageID != 0 {
			edit := newEdit(chatID, messageID, text)
			if kb != nil {
				edit.ReplyMarkup = kb
			}
			return h.send(edit)
		}

		msg := newMessage(chatID, text)
		if kb != nil {
			msg.ReplyMarkup = *kb
		}
		return h.send(msg)
	}
}

// handleRandom shows random name from today list (guided) OR any name (free).
func (h *Handler) handleRandom(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			settings = entities.NewUserSettings(userID)
		}

		var nameNumbers []int

		if settings.LearningMode == "guided" {
			// Guided: random from today's names
			todayNames, err := h.dailyNameService.GetTodayNames(ctx, userID)
			if err != nil || len(todayNames) == 0 {
				msg := newPlainMessage(chatID, "📚 Сегодня ещё не начали изучение.\nИспользуйте /next!")
				return h.send(msg)
			}
			nameNumbers = todayNames
		} else {
			// Free: truly random from all 99
			name, err := h.nameService.GetRandom(ctx)
			if err != nil {
				h.logger.Error("failed to get random name", zap.Error(err))
				msg := newPlainMessage(chatID, msgNameUnavailable)
				return h.send(msg)
			}

			msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (*entities.Name, error) {
				return h.nameService.GetByNumber(ctx, name.Number)
			}, chatID)
			if err != nil {
				return err
			}

			if err = h.send(msg); err != nil {
				return err
			}
			if audio != nil {
				_ = h.send(*audio)
			}
			return nil
		}

		// Guided: pick random from today names
		randomIndex := rand.Intn(len(nameNumbers))
		nameNumber := nameNumbers[randomIndex]

		msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (*entities.Name, error) {
			return h.nameService.GetByNumber(ctx, nameNumber)
		}, chatID)
		if err != nil {
			return err
		}

		if audio != nil {
			_ = h.send(*audio)
		}
		if err = h.send(msg); err != nil {
			return err
		}

		return nil
	}
}

// handleAll sends a paginated list of all names.
func (h *Handler) handleAll() HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		names, err := h.getAllNames(ctx)
		if err != nil {
			return err
		}

		if names == nil {
			msg := newPlainMessage(chatID, msgNameUnavailable)
			return h.send(msg)
		}

		page := 0
		text, totalPages := buildNamesPage(names, page)
		prevData := buildNameCallback(page - 1)
		nextData := buildNameCallback(page + 1)

		msg := newMessage(chatID, text)
		kb := buildNameKeyboard(page, totalPages, prevData, nextData)
		if kb != nil {
			msg.ReplyMarkup = *kb
		}

		return h.send(msg)
	}
}

// handleRangeNumbers sends a paginated list of names in a specified range.
func (h *Handler) handleRangeNumbers(from, to int) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		if from < 1 || to > 99 || from > to {
			return h.send(newPlainMessage(chatID, msgInvalidRange))
		}

		names, err := h.getAllNames(ctx)
		if err != nil {
			return err
		}
		if names == nil {
			return h.send(newPlainMessage(chatID, msgNameUnavailable))
		}

		pages := buildRangePages(names, from, to)
		if len(pages) == 0 {
			return h.send(newPlainMessage(chatID, msgNameUnavailable))
		}

		page := 0
		totalPages := len(pages)
		prevData := buildRangeCallback(page-1, from, to)
		nextData := buildRangeCallback(page+1, from, to)

		msg := newMessage(chatID, pages[page])
		kb := buildNameKeyboard(page, totalPages, prevData, nextData)
		if kb != nil {
			msg.ReplyMarkup = *kb
		}

		return h.send(msg)
	}
}

// handleProgress displays user progress.
func (h *Handler) handleProgress(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		h.logger.Debug("rendering progress", zap.Int64("user_id", userID))

		text, keyboard, err := h.RenderProgress(ctx, userID, true)
		if err != nil {
			h.logger.Error("failed to render progress",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			msg := newPlainMessage(chatID, msgProgressUnavailable)
			return h.send(msg)
		}

		msg := newMessage(chatID, text)
		if keyboard != nil {
			msg.ReplyMarkup = *keyboard
		}

		return h.send(msg)
	}
}

// handleSettings displays user settings.
func (h *Handler) handleSettings(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		h.logger.Debug("rendering settings", zap.Int64("user_id", userID))

		text, keyboard, err := h.RenderSettings(ctx, userID)
		if err != nil {
			h.logger.Error("failed to render settings",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			msg := newPlainMessage(chatID, msgSettingsUnavailable)
			return h.send(msg)
		}

		msg := newMessage(chatID, text)
		msg.ReplyMarkup = keyboard
		return h.send(msg)
	}
}

// handleQuiz starts a quiz for the user.
func (h *Handler) handleQuiz(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		isFirstQuiz, err := h.quizService.IsFirstQuiz(ctx, userID)
		if err != nil {
			return err
		}

		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			h.logger.Error("failed to get settings for quiz",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			msg := newPlainMessage(chatID, msgQuizUnavailable)
			return h.send(msg)
		}

		// Check for active session
		activeSession, err := h.quizService.GetActiveSession(ctx, userID)
		if err != nil {
			h.logger.Error("failed to get active session",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			return h.send(newPlainMessage(chatID, msgQuizUnavailable))
		}

		// If there's an active session, resume it
		if activeSession != nil && activeSession.SessionStatus == "active" {
			// Delete previous quiz question if it exists
			if oldMsgID, exists := h.quizStorage.GetMessageID(activeSession.ID); exists {
				_, _ = h.bot.Send(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
			}

			q, name, err := h.quizService.GetCurrentQuestion(ctx, activeSession.ID, activeSession.CurrentQuestionNum)
			if err != nil {
				h.logger.Error("failed to get current question for resume",
					zap.Int64("session_id", activeSession.ID),
					zap.Int("question_num", activeSession.CurrentQuestionNum),
					zap.Error(err),
				)
				return h.send(newPlainMessage(chatID, msgQuizUnavailable))
			}

			_ = h.send(newMessage(chatID, md("📝 Продолжаем квиз...")))
			return h.sendQuizQuestionFromDB(chatID, activeSession, q, name, activeSession.CurrentQuestionNum, isFirstQuiz)
		}

		// Start new quiz session
		totalQuestions := 5 // Default number of questions
		h.logger.Debug("starting new quiz session",
			zap.Int64("user_id", userID),
			zap.Int("total_questions", totalQuestions),
			zap.String("quiz_mode", settings.QuizMode),
		)

		session, names, err := h.quizService.StartQuizSession(ctx, userID, totalQuestions)
		if err != nil {
			h.logger.Error("failed to start quiz session",
				zap.Int64("user_id", userID),
				zap.String("quiz_mode", settings.QuizMode),
				zap.Error(err),
			)

			if errors.Is(err, service.ErrNoQuestionsAvailable) {
				stats, stErr := h.progressService.GetProgressSummary(ctx, userID)
				if stErr == nil && stats != nil && stats.Learned >= 99 {
					return h.send(newMessage(chatID, msgNoNewNames()))
				}

				if settings.LearningMode == string(entities.ModeGuided) && settings.QuizMode == "new" {
					return h.send(newMessage(chatID,
						md("🆕 Новых вопросов нет.\n\n")+
							md("В Guided режиме «Новые» — это только незавершённые имена из /today.\n")+
							md("Если всё выучено — дождитесь следующего дня или увеличьте «имён в день» в /settings."),
					))
				}

				switch settings.QuizMode {
				case "review":
					return h.send(newMessage(chatID, msgNoReviews()))
				case "new":
					return h.send(newMessage(chatID, msgNoAvailableQuestions()))
				default:
					return h.send(newMessage(chatID, msgNoAvailableQuestions()))
				}
			}
			return h.send(newPlainMessage(chatID, msgQuizUnavailable))
		}

		h.logger.Debug("quiz session created",
			zap.Int64("session_id", session.ID),
			zap.Int("names_count", len(names)),
		)

		// Store names for quick access during quiz
		h.quizStorage.Store(session.ID, names)

		if err := h.send(newMessage(chatID, buildQuizStartMessage(settings.QuizMode))); err != nil {
			return err
		}

		q, name, err := h.quizService.GetCurrentQuestion(ctx, session.ID, 1)
		if err != nil {
			h.logger.Error("failed to get first question", zap.Int64("session_id", session.ID), zap.Error(err))
			return h.send(newPlainMessage(chatID, msgQuizUnavailable))
		}

		return h.sendQuizQuestionFromDB(chatID, session, q, name, 1, isFirstQuiz)
	}
}

func (h *Handler) handleReset() HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		text := md("⚠️ ") + bold("Сброс прогресса и настроек") + "\n\n" +
			md("Вы точно хотите сбросить прогресс?") + "\n" +
			md("Вы потеряете все изученные имена, дневной план и статистику.") + "\n\n" +
			md("Это действие нельзя отменить.")

		msg := newMessage(chatID, text)
		if kb := buildResetKeyboard(); kb != nil {
			msg.ReplyMarkup = *kb
		}
		return h.send(msg)
	}
}
