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

		if err = h.send(msg); err != nil {
			return err
		}

		if audio != nil {
			_ = h.send(*audio)
		}

		return nil
	}
}

// handleNext shows today's names or introduces a new one if quota allows.
func (h *Handler) handleNext(userID int64, messageID int) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			settings = entities.NewUserSettings(userID)
		}

		namesPerDay := settings.NamesPerDay
		if namesPerDay <= 0 {
			namesPerDay = 1
		}

		// Get today's introduced names
		todayNames, err := h.dailyNameService.GetTodayNames(ctx, userID)
		if err != nil {
			h.logger.Error("failed to get today names", zap.Error(err))
			msg := newPlainMessage(chatID, msgNameUnavailable)
			return h.send(msg)
		}

		var name *entities.Name

		// If we have names for today, show one of them
		if len(todayNames) > 0 {
			// Show the first name from today's list
			name, err = h.nameService.GetByNumber(ctx, todayNames[0])
			if err != nil {
				h.logger.Error("failed to get name by number", zap.Error(err))
				msg := newPlainMessage(chatID, msgNameUnavailable)
				return h.send(msg)
			}

			h.logger.Info("showing today's name",
				zap.Int64("user_id", userID),
				zap.Int("name_number", name.Number),
				zap.Int("today_count", len(todayNames)),
				zap.Int("names_per_day", namesPerDay),
			)
		} else {
			// No names for today, check if we can introduce a new one
			todayCount := len(todayNames)
			if todayCount >= namesPerDay {
				msg := newPlainMessage(chatID,
					fmt.Sprintf("📚 Сегодня вы уже изучаете %d имя(ён).\n\n"+
						"Пройдите /quiz чтобы закрепить текущие имена и разблокировать следующие!\n\n"+
						"💡 Или увеличьте лимит в /settings → Имён в день",
						namesPerDay))
				return h.send(msg)
			}

			// Get next name for introduction
			nameNumbers, err := h.progressService.GetNewNames(ctx, userID, 1)
			if err != nil {
				h.logger.Error("failed to get new names", zap.Error(err))
				msg := newPlainMessage(chatID, msgNameUnavailable)
				return h.send(msg)
			}

			if len(nameNumbers) == 0 {
				msg := newPlainMessage(chatID, "🎉 Вы уже начали изучение всех 99 имён!\n\nИспользуйте /quiz для повторения.")
				return h.send(msg)
			}

			name, err = h.nameService.GetByNumber(ctx, nameNumbers[0])
			if err != nil {
				h.logger.Error("failed to get name by number", zap.Error(err))
				msg := newPlainMessage(chatID, msgNameUnavailable)
				return h.send(msg)
			}

			// Mark as introduced in progress
			if err = h.progressService.IntroduceName(ctx, userID, name.Number); err != nil {
				h.logger.Error("failed to mark name as introduced", zap.Error(err))
				msg := newPlainMessage(chatID, "❌ Ошибка при сохранении прогресса")
				return h.send(msg)
			}

			// Add to today's names
			if err = h.dailyNameService.AddTodayName(ctx, userID, name.Number); err != nil {
				h.logger.Error("failed to add today name", zap.Error(err))
			}

			h.logger.Info("introduced new name",
				zap.Int64("user_id", userID),
				zap.Int("name_number", name.Number),
				zap.Int("today_count", todayCount+1),
				zap.Int("names_per_day", namesPerDay),
			)
		}

		msg, audio, err := buildNameResponse(ctx, func(ctx context.Context) (*entities.Name, error) {
			return h.nameService.GetByNumber(ctx, name.Number)
		}, chatID)
		if err != nil {
			return err
		}

		// Delete the /next command message
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

		if err = h.send(msg); err != nil {
			return err
		}

		if audio != nil {
			_ = h.send(*audio)
		}

		return nil
	}
}

// handleToday shows all names introduced today.
func (h *Handler) handleToday(userID int64, messageID int) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			settings = entities.NewUserSettings(userID)
		}

		todayNames, err := h.dailyNameService.GetTodayNames(ctx, userID)
		if err != nil {
			h.logger.Error("failed to get today names", zap.Error(err))
			msg := newPlainMessage(chatID, "❌ Ошибка загрузки списка")
			return h.send(msg)
		}

		if len(todayNames) == 0 {
			msg := newPlainMessage(chatID,
				fmt.Sprintf("📚 Сегодня ещё не начали изучение.\n\n"+
					"💡 Используйте /next чтобы начать! (лимит: %d в день)",
					settings.NamesPerDay))
			return h.send(msg)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📚 *Сегодня изучаете (%d/%d):*\n\n",
			len(todayNames), settings.NamesPerDay))

		for i, nameNumber := range todayNames {
			name, err := h.nameService.GetByNumber(ctx, nameNumber)
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("%d️⃣ %s\n", i+1, name.Translation))
		}

		if len(todayNames) < settings.NamesPerDay {
			sb.WriteString(fmt.Sprintf("\n✅ *%d/%d* — пройдите /quiz чтобы разблокировать следующие!",
				len(todayNames), settings.NamesPerDay))
		}

		// Delete the /today command message
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

		msg := newMessage(chatID, sb.String())
		return h.send(msg)
	}
}

// handleRandom shows random name from today list (guided) OR any name (free).
func (h *Handler) handleRandom(userID int64, messageID int) HandlerFunc {
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

			// Delete the /random command message
			deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
			_, _ = h.bot.Send(deleteCmd)

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

		// Delete the /random command message
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

		if err = h.send(msg); err != nil {
			return err
		}
		if audio != nil {
			_ = h.send(*audio)
		}

		return nil
	}
}

// handleAll sends a paginated list of all names.
func (h *Handler) handleAll(messageID int) HandlerFunc {
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

		// Delete the /all command message
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

		msg := newMessage(chatID, text)
		kb := buildNameKeyboard(page, totalPages, prevData, nextData)
		if kb != nil {
			msg.ReplyMarkup = *kb
		}

		return h.send(msg)
	}
}

// handleRange sends a paginated list of names in a specified range.
func (h *Handler) handleRange(argsStr string, messageID int) HandlerFunc {
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

		// Delete the /range command message
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

		msg := newMessage(chatID, pages[page])
		kb := buildNameKeyboard(page, totalPages, prevData, nextData)
		if kb != nil {
			msg.ReplyMarkup = *kb
		}

		return h.send(msg)
	}
}

// handleProgress displays user progress.
func (h *Handler) handleProgress(userID int64, messageID int) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		h.logger.Debug("rendering progress", zap.Int64("user_id", userID))

		// Delete the /progress command message
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

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
func (h *Handler) handleSettings(userID int64, messageID int) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		h.logger.Debug("rendering settings", zap.Int64("user_id", userID))

		// Delete the /settings command message
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

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
func (h *Handler) handleQuiz(userID int64, messageID int) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
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
				deleteMsg := tgbotapi.NewDeleteMessage(chatID, oldMsgID)
				_, _ = h.bot.Send(deleteMsg)
			}

			// Delete the /quiz command message
			deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
			_, _ = h.bot.Send(deleteCmd)

			names := h.quizStorage.Get(activeSession.ID)
			if len(names) > 0 {
				// Resume from current question
				question, name, err := h.quizService.GetCurrentQuestion(ctx, activeSession.ID, activeSession.CurrentQuestionNum)
				if err != nil {
					h.logger.Error("failed to get current question for resume",
						zap.Int64("session_id", activeSession.ID),
						zap.Int("question_num", activeSession.CurrentQuestionNum),
						zap.Error(err),
					)
					return h.send(newPlainMessage(chatID, msgQuizUnavailable))
				}

				resumeMsg := newMessage(chatID, md("📝 Продолжаем квиз..."))
				if err := h.send(resumeMsg); err != nil {
					return err
				}

				return h.sendQuizQuestionFromDB(chatID, activeSession, question, name, activeSession.CurrentQuestionNum)
			}
		}

		// Delete the /quiz command message for new quiz
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = h.bot.Send(deleteCmd)

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
				switch settings.QuizMode {
				case "review":
					return h.send(newMessage(chatID, msgNoReviews()))
				case "new":
					return h.send(newMessage(chatID, msgNoNewNames()))
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

		// Send start message
		startMsg := newMessage(chatID, buildQuizStartMessage(settings.QuizMode))
		if err := h.send(startMsg); err != nil {
			return err
		}

		// Get first question and send it
		question, name, err := h.quizService.GetCurrentQuestion(ctx, session.ID, 1)
		if err != nil {
			h.logger.Error("failed to get first question",
				zap.Int64("session_id", session.ID),
				zap.Error(err),
			)
			return h.send(newPlainMessage(chatID, msgQuizUnavailable))
		}

		return h.sendQuizQuestionFromDB(chatID, session, question, name, 1)
	}
}
