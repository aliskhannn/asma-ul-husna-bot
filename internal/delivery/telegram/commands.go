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
		}, chatID, "", "")
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

// handleNext shows one name in phase new introduced today.
func (h *Handler) handleNext(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil || settings == nil {
			settings = entities.NewUserSettings(userID)
		}

		namesPerDay := settings.NamesPerDay
		if namesPerDay <= 0 {
			namesPerDay = 1
		}

		stats, err := h.progressService.GetProgressSummary(ctx, userID)
		if err != nil {
			return h.send(newPlainMessage(chatID, msgInternalError))
		}
		learnedTotal := stats.Learned

		isFirstTime := stats.DueToday == 0 && stats.Learned == 0 && stats.InProgress == 0 && stats.NotStarted == 99
		if isFirstTime {
			if err := h.send(newMessage(chatID, nextFirstTimeIntroMessage(namesPerDay))); err != nil {
				return h.send(newPlainMessage(chatID, msgInternalError))
			}
		}

		introducedToday, err := h.progressService.CountIntroducedToday(ctx, userID, settings.Timezone)
		if err != nil {
			return h.send(newPlainMessage(chatID, msgInternalError))
		}

		if introducedToday >= namesPerDay {
			return h.sendNextLimitReached(chatID, introducedToday, namesPerDay)
		}

		suffix := nextHintMessage()

		// 1) Guided debt first
		if settings.LearningMode == string(entities.ModeGuided) {
			hasDebt, err := h.dailyNameService.HasUnfinishedDays(ctx, userID)
			if err != nil {
				h.logger.Error("failed to check unfinished days", zap.Error(err))
				return h.send(newPlainMessage(chatID, msgNameUnavailable))
			}
			if hasDebt {
				n, err := h.dailyNameService.GetOldestUnfinishedName(ctx, userID)
				if err != nil {
					h.logger.Error("failed to get oldest unfinished name", zap.Error(err))
					return h.send(newPlainMessage(chatID, msgNameUnavailable))
				}

				prefix := buildNextPrefix(introducedToday, namesPerDay, learnedTotal)
				return h.sendNameCardWithPrefix(ctx, chatID, 0, "next", n, prefix, suffix)
			}
		}

		// 2) today plan
		todayNames, err := h.dailyNameService.GetTodayNames(ctx, userID)
		if err != nil {
			h.logger.Error("failed to get today names", zap.Error(err))
			return h.send(newPlainMessage(chatID, msgNameUnavailable))
		}

		// Helper: introduce one + recount + send card (so prefix is correct)
		introduceAndShow := func() error {
			nameNumber, err := h.introduceOne(ctx, userID, chatID, "", "")
			if err != nil {
				h.logger.Error("failed to introduce name", zap.Error(err))
				return err
			}

			introducedToday2, err := h.progressService.CountIntroducedToday(ctx, userID, settings.Timezone)
			if err != nil {
				return h.send(newPlainMessage(chatID, msgInternalError))
			}

			prefix := buildNextPrefix(introducedToday2, namesPerDay, learnedTotal)
			return h.sendNameCardWithPrefix(ctx, chatID, 0, "next", nameNumber, prefix, suffix)
		}

		// 3) empty => introduce (if quota allows)
		if len(todayNames) == 0 {
			return introduceAndShow()
		}

		dec, err := h.decideToday(ctx, userID, todayNames)
		if err != nil {
			h.logger.Error("failed to decide today state", zap.Error(err))
			return h.send(newPlainMessage(chatID, msgNameUnavailable))
		}

		planFull := len(todayNames) >= namesPerDay

		// If there is new to show => show (prefix based on introducedToday we already computed)
		if dec.State == TodayHasNew || dec.HasNew {
			prefix := buildNextPrefix(introducedToday, namesPerDay, learnedTotal)
			return h.sendNameCardWithPrefix(ctx, chatID, 0, "next", dec.NewToShow, prefix, suffix)
		}

		// If plan not full => introduce one and show it with updated prefix
		if !planFull {
			return introduceAndShow()
		}

		// planFull && hasLearning => blocked
		if planFull && dec.HasLearning {
			return h.sendNextBlockedNeedQuiz(chatID, namesPerDay)
		}

		switch dec.State {
		case TodayHasNew:
			prefix := buildNextPrefix(introducedToday, namesPerDay, learnedTotal)
			return h.sendNameCardWithPrefix(ctx, chatID, 0, "next", dec.NewToShow, prefix, suffix)

		case TodayAllLearning:
			return h.sendNextBlockedNeedQuiz(chatID, namesPerDay)

		case TodayMixed:
			if n, ok := h.chooseFirstNotMastered(ctx, userID, todayNames); ok {
				prefix := buildNextPrefix(introducedToday, namesPerDay, learnedTotal)
				return h.sendNameCardWithPrefix(ctx, chatID, 0, "next", n, prefix, suffix)
			}
			return h.send(newPlainMessage(chatID, msgNameUnavailable))

		case TodayAllMastered:
			if len(todayNames) < namesPerDay {
				return introduceAndShow()
			}
			return h.send(newPlainMessage(chatID,
				fmt.Sprintf("✅ Сегодняшний план выполнен (%d/%d).\n\n"+
					"Чтобы открыть новые имена:\n"+
					"• Увеличьте «имён в день» в /settings\n"+
					"• Или дождитесь следующего дня",
					len(todayNames), namesPerDay)))

		default:
			return h.send(newPlainMessage(chatID, msgNameUnavailable))
		}
	}
}

// introduceOne introduces one new name and adds it to today's plan.
func (h *Handler) introduceOne(ctx context.Context, userID, chatID int64, prefix, suffix string) (int, error) {
	newNums, err := h.progressService.GetNewNames(ctx, userID, 1)
	if err != nil {
		h.logger.Error("failed to get new names", zap.Error(err))
		return 0, h.send(newPlainMessage(chatID, msgNameUnavailable))
	}
	if len(newNums) == 0 {
		return 0, h.send(newPlainMessage(chatID,
			"🎉 Вы уже начали изучение всех 99 имён!\n\nИспользуйте /quiz для повторения."))
	}

	newNum := newNums[0]

	if err := h.progressService.IntroduceName(ctx, userID, newNum); err != nil {
		h.logger.Error("failed to mark name as introduced", zap.Error(err))
		return 0, h.send(newPlainMessage(chatID, "❌ Ошибка при сохранении прогресса"))
	}
	if err := h.dailyNameService.AddTodayName(ctx, userID, newNum); err != nil {
		h.logger.Error("failed to add today name", zap.Error(err))
		return 0, h.send(newPlainMessage(chatID, "❌ Ошибка при сохранении дневного плана"))
	}

	return newNum, nil
}

type TodayState int

const (
	TodayEmpty TodayState = iota
	TodayAllLearning
	TodayHasNew
	TodayAllMastered
	TodayMixed
)

type TodayDecision struct {
	State       TodayState
	NewToShow   int // nameNumber if State == TodayHasNew
	HasLearning bool
	HasNew      bool
	AllMastered bool
}

func (h *Handler) decideToday(ctx context.Context, userID int64, todayNames []int) (TodayDecision, error) {
	if len(todayNames) == 0 {
		return TodayDecision{State: TodayEmpty}, nil
	}

	m, err := h.progressService.GetByNumbers(ctx, userID, todayNames)
	if err != nil {
		return TodayDecision{}, err
	}

	dec := TodayDecision{
		State:       TodayMixed,
		NewToShow:   0,
		HasLearning: false,
		HasNew:      false,
		AllMastered: true,
	}

	allLearning := true

	for _, num := range todayNames {
		p, ok := m[num]
		phase := string(entities.PhaseNew)
		if ok && p != nil {
			phase = string(p.Phase)
		}

		if phase == string(entities.PhaseNew) && !dec.HasNew {
			dec.HasNew = true
			dec.NewToShow = num
		}
		if phase == string(entities.PhaseLearning) {
			dec.HasLearning = true
		}

		if phase != string(entities.PhaseMastered) {
			dec.AllMastered = false
		}
		if phase != string(entities.PhaseLearning) {
			allLearning = false
		}
	}

	switch {
	case dec.HasNew:
		dec.State = TodayHasNew
	case dec.AllMastered:
		dec.State = TodayAllMastered
	case allLearning:
		dec.State = TodayAllLearning
	default:
		dec.State = TodayMixed
	}

	return dec, nil
}

// handleToday shows all names introduced today.
func (h *Handler) handleToday(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil || settings == nil {
			settings = entities.NewUserSettings(userID)
		}

		namesPerDay := settings.NamesPerDay
		if namesPerDay <= 0 {
			namesPerDay = 1
		}

		todayNames, err := h.dailyNameService.GetTodayNames(ctx, userID)
		if err != nil {
			h.logger.Error("failed to get today names", zap.Error(err))
			return h.send(newPlainMessage(chatID, "❌ Ошибка загрузки списка"))
		}

		// Empty today plan
		if len(todayNames) == 0 {
			msg := newPlainMessage(chatID,
				fmt.Sprintf("📚 Сегодня ещё не начали изучение.\n\n"+
					"💡 Используйте /next чтобы начать! (лимит: %d в день)",
					namesPerDay))
			return h.send(msg)
		}

		// Build the list with learning status
		return h.sendTodayList(ctx, chatID, userID, settings, todayNames)
	}
}

func (h *Handler) chooseFirstUnfinished(ctx context.Context, userID int64, nums []int) (int, bool) {
	for _, n := range nums {
		streak, err := h.progressService.GetStreak(ctx, userID, n)
		if err != nil {
			return n, true
		}
		if streak < 7 {
			return n, true
		}
	}
	return 0, false
}

func (h *Handler) chooseFirstNotMastered(ctx context.Context, userID int64, todayNames []int) (int, bool) {
	if len(todayNames) == 0 {
		return 0, false
	}

	m, err := h.progressService.GetByNumbers(ctx, userID, todayNames)
	if err != nil {
		h.logger.Warn("failed to load progress for today names", zap.Error(err))
		return 0, false
	}

	for _, num := range todayNames {
		p, ok := m[num]
		if !ok || p == nil {
			// нет записи прогресса => считаем что это точно не mastered
			return num, true
		}
		if p.Phase != entities.PhaseMastered {
			return num, true
		}
	}

	return 0, false
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
			}, chatID, "", "")
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
		}, chatID, "", "")
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

// handleRange sends a paginated list of names in a specified range.
func (h *Handler) handleRange(argsStr string) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		args := strings.Fields(argsStr)
		if len(args) != 2 {
			return h.send(newPlainMessage(chatID, msgUseRange))
		}

		from, errFrom := strconv.Atoi(args[0])
		to, errTo := strconv.Atoi(args[1])
		if errFrom != nil || errTo != nil || from < 1 || to > 99 || from > to {
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
			return h.sendQuizQuestionFromDB(ctx, userID, chatID, activeSession, q, name, activeSession.CurrentQuestionNum, isFirstQuiz)
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

		return h.sendQuizQuestionFromDB(ctx, userID, chatID, session, q, name, 1, isFirstQuiz)
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
