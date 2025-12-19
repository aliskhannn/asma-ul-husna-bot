package telegram

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

// handleNumber handles numeric input (name by number).
func (h *Handler) handleNumber(numStr string, userID int64) HandlerFunc {
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

		if err = h.progressService.MarkAsViewed(ctx, userID, n); err != nil {
			return err
		}

		return nil
	}
}

// handleRandom handles /random command.
func (h *Handler) handleRandom(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		name, err := h.nameService.GetRandom(ctx)
		if err != nil {
			return err
		}

		msg := newMessage(chatID, formatNameMessage(name))
		if err = h.send(msg); err != nil {
			return err
		}

		if name.Audio != "" {
			audio := buildNameAudio(name, chatID)
			if err = h.send(*audio); err != nil {
				return err
			}
		}

		if err = h.progressService.MarkAsViewed(ctx, userID, name.Number); err != nil {
			return err
		}

		return nil
	}
}

// handleAll handles /all command.
func (h *Handler) handleAll(ctx context.Context, chatID int64) error {
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

// handleRange handles /range command.
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

// handleProgress handles /progress command.
func (h *Handler) handleProgress(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		text, keyboard, err := h.RenderProgress(ctx, userID, true)
		if err != nil {
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

// handleSettings handles /settings command.
func (h *Handler) handleSettings(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		text, keyboard, err := h.RenderSettings(ctx, userID)
		if err != nil {
			msg := newPlainMessage(chatID, msgSettingsUnavailable)
			return h.send(msg)
		}

		msg := newMessage(chatID, text)
		msg.ReplyMarkup = keyboard
		return h.send(msg)
	}
}

// quizHandler handles /quiz command.
func (h *Handler) handleQuiz(userID int64) HandlerFunc {
	return func(ctx context.Context, chatID int64) error {
		settings, err := h.settingsService.GetOrCreate(ctx, userID)
		if err != nil {
			msg := newPlainMessage(chatID, msgSettingsUnavailable)
			return h.send(msg)
		}

		mode := settings.QuizMode
		session, questions, err := h.quizService.GenerateQuiz(ctx, userID, mode)
		if err != nil {
			if errors.Is(err, service.ErrNoQuestionsAvailable) {
				switch mode {
				case "review":
					return h.send(newPlainMessage(chatID, msgNoReviews))
				case "new":
					return h.send(newPlainMessage(chatID, msgNoNewNames))
				default:
					return h.send(newPlainMessage(chatID, msgNoAvailableQuestions))
				}
			}

			return h.send(newPlainMessage(chatID, msgQuizUnavailable))
		}

		h.storeQuizQuestions(session.ID, questions)

		startMsg := newMessage(chatID, buildQuizStartMessage(mode))
		if err := h.send(startMsg); err != nil {
			return err
		}

		return h.sendQuizQuestion(chatID, session, &questions[0], 1)
	}
}
