package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	data := cb.Data

	switch {
	case strings.HasPrefix(data, "range:"):
		h.withCallbackErrorHandling(h.rangeCallbackHandler)(ctx, cb)

	case strings.HasPrefix(data, "name:"):
		h.withCallbackErrorHandling(h.allCallbackHandler)(ctx, cb)

	case strings.HasPrefix(data, "settings:"):
		h.withCallbackErrorHandling(h.settingsCallbackHandler)(ctx, cb)

	default:
		h.logger.Warn("unknown callback data",
			zap.String("data", data),
		)
		return
	}

	// Remove the user's "clock".
	answer := tgbotapi.NewCallback(cb.ID, "")
	if _, err := h.bot.Request(answer); err != nil {
		h.logger.Error("callback answer error",
			zap.Error(err),
			zap.String("data", cb.Data),
		)
	}
}

func (h *Handler) allCallbackHandler(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	pageStr := strings.TrimPrefix(cb.Data, "name:")
	page, err := strconv.Atoi(pageStr)
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
		msg := newHTMLMessage(cb.Message.Chat.ID, msgNameUnavailable)
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

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		edit.ReplyMarkup = kb
	}

	return h.send(edit)
}

func (h *Handler) rangeCallbackHandler(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 4 {
		h.logger.Warn("invalid range callback data",
			zap.String("data", cb.Data),
		)
		return nil
	}

	page, err1 := strconv.Atoi(parts[1])
	from, err2 := strconv.Atoi(parts[2])
	to, err3 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil || err3 != nil || page < 0 || from < 1 || to > 99 || from > to {
		h.logger.Warn("invalid range callback values",
			zap.String("data", cb.Data),
			zap.Error(err1),
			zap.Error(err2),
			zap.Error(err3),
		)
		return nil
	}

	names, err := h.getAllNames(ctx)
	if err != nil {
		return err
	}
	if names == nil {
		msg := newHTMLMessage(cb.Message.Chat.ID, msgNameUnavailable)
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

	prevData := fmt.Sprintf("range:%d:%d:%d", page-1, from, to)
	nextData := fmt.Sprintf("range:%d:%d:%d", page+1, from, to)

	kb := buildNameKeyboard(page, totalPages, prevData, nextData)

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		edit.ReplyMarkup = kb
	}

	return h.send(edit)
}

func (h *Handler) settingsCallbackHandler(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	parts := strings.Split(cb.Data, ":")
	// "settings:<key>" or "settings:<key>:<value>"
	if len(parts) < 2 {
		h.logger.Warn("invalid settings callback", zap.String("data", cb.Data))
		return nil
	}

	key := parts[1]

	// "settings:<key>"
	if len(parts) == 2 {
		switch key {
		case "menu":
			return h.showSettingsMenu(ctx, cb)
		case "names_per_day":
			return h.showNamesPerDayMenu(cb)
		case "quiz_length":
			return h.showQuizLengthMenu(cb)
		case "quiz_mode":
			return h.showQuizModesMenu(cb)
		case "toggle_transliteration":
			return h.showToggleTransliterationMenu(cb)
		case "toggle_audio":
			return h.showToggleAudioMenu(cb)
		default:
			h.logger.Warn("unknown settings key", zap.String("key", key), zap.String("data", cb.Data))
			return nil
		}
	}

	// "settings:<key>:<value>"
	value := parts[2]

	switch key {
	case "names_per_day":
		return h.applyNamesPerDay(ctx, cb, value)
	case "quiz_length":
		return h.applyQuizLength(ctx, cb, value)
	case "quiz_mode":
		return h.applyQuizMode(ctx, cb, value)
	case "toggle_transliteration":
		return h.applyToggleTransliteration(ctx, cb, value)
	case "toggle_audio":
		return h.applyToggleAudio(ctx, cb, value)
	default:
		h.logger.Warn("unknown settings key with value", zap.String("key", key), zap.String("data", cb.Data))
		return nil
	}
}

func (h *Handler) showSettingsMenu(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	userID := cb.From.ID

	settings, err := h.settingsService.GetOrCreate(ctx, userID)
	if err != nil {
		msg := newHTMLMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
		return h.send(msg)
	}

	text := fmt.Sprintf(
		"<b>⚙️ Настройки</b>\n\n"+
			"📚 <b>Имён в день:</b> %d\n"+
			"📝 <b>Длина квиза:</b> %d\n"+
			"🎲 <b>Режим квиза:</b> %s\n"+
			"🔤 <b>Транслитерация:</b> %s\n"+
			"🔊 <b>Аудио:</b> %s\n",
		settings.NamesPerDay,
		settings.QuizLength,
		formatQuizMode(settings.QuizMode),
		formatBool(settings.ShowTransliteration),
		formatBool(settings.ShowAudio),
	)

	kb := buildSettingsKeyboard()

	edit := tgbotapi.NewEditMessageText(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		text,
	)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = &kb

	return h.send(edit)
}

func (h *Handler) showNamesPerDayMenu(cb *tgbotapi.CallbackQuery) error {
	message := "📚 *Сколько новых имён изучать в день?*\n\n" +
		"Выберите интенсивность обучения:"

	kb := buildNamesPerDayKeyboard()

	edit := tgbotapi.NewEditMessageText(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		message,
	)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &kb

	return h.send(edit)
}

func (h *Handler) showQuizLengthMenu(cb *tgbotapi.CallbackQuery) error {
	message := "📝 *Длина квиза*\n\n" +
		"Сколько вопросов должно быть в одном квизе?"

	kb := buildQuizLengthKeyboard()

	edit := tgbotapi.NewEditMessageText(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		message,
	)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &kb

	return h.send(edit)
}

func (h *Handler) showQuizModesMenu(cb *tgbotapi.CallbackQuery) error {
	message := "🎲 *Режим квиза*\n\n" +
		"Выберите, какие имена включать в квиз: только новые, только на повторение или оба варианта."

	kb := buildQuizModesKeyboard()

	edit := tgbotapi.NewEditMessageText(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		message,
	)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &kb

	return h.send(edit)
}

func (h *Handler) showToggleTransliterationMenu(cb *tgbotapi.CallbackQuery) error {
	message := "🔤 *Транслитерация*\n\n" +
		"Показывать ли транслитерацию арабских имён латиницей?"

	kb := buildToggleTransliterationKeyboard()

	edit := tgbotapi.NewEditMessageText(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		message,
	)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &kb

	return h.send(edit)
}

func (h *Handler) showToggleAudioMenu(cb *tgbotapi.CallbackQuery) error {
	message := "🔊 *Аудио*\n\n" +
		"Включить или отключить отправку аудиопроизношения имён."

	kb := buildToggleAudioKeyboard()

	edit := tgbotapi.NewEditMessageText(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		message,
	)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &kb

	return h.send(edit)
}

func (h *Handler) applyNamesPerDay(ctx context.Context, cb *tgbotapi.CallbackQuery, value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		h.logger.Warn("invalid names_per_day value",
			zap.String("data", cb.Data),
			zap.Error(err),
		)
		return nil
	}

	if v < 1 || v > 20 {
		h.logger.Warn("names_per_day value out of range",
			zap.Int("value", v),
			zap.String("data", cb.Data),
		)
		return nil
	}

	if err := h.settingsService.UpdateNamesPerDay(ctx, cb.From.ID, v); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newHTMLMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}

		return err
	}

	confirm := tgbotapi.NewCallback(cb.ID, fmt.Sprintf("Имён в день: %d", v))
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send names_per_day confirmation",
			zap.Error(err),
			zap.Int("value", v),
		)
	}

	return h.showSettingsMenu(ctx, cb)
}

func (h *Handler) applyQuizLength(ctx context.Context, cb *tgbotapi.CallbackQuery, value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		h.logger.Warn("invalid quiz_length value",
			zap.String("data", cb.Data),
			zap.Error(err),
		)
		return nil
	}

	if v < 5 || v > 50 {
		h.logger.Warn("quiz_length value out of range",
			zap.Int("value", v),
			zap.String("data", cb.Data),
		)
		return nil
	}

	if err := h.settingsService.UpdateQuizLength(ctx, cb.From.ID, v); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newHTMLMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}

		return err
	}

	confirm := tgbotapi.NewCallback(cb.ID, fmt.Sprintf("Длина квиза: %d", v))
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send quiz_length confirmation",
			zap.Error(err),
			zap.Int("value", v),
		)
	}

	return h.showSettingsMenu(ctx, cb)
}

func (h *Handler) applyQuizMode(ctx context.Context, cb *tgbotapi.CallbackQuery, value string) error {
	quizMode := value // "new_only", "review_only", "mixed"

	if err := h.settingsService.UpdateQuizMode(ctx, cb.From.ID, quizMode); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newHTMLMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}
		return err
	}

	confirm := tgbotapi.NewCallback(cb.ID,
		fmt.Sprintf("Режим квиза: %s", formatQuizMode(quizMode)),
	)
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send quiz_mode confirmation",
			zap.Error(err),
			zap.String("quiz_mode", quizMode),
		)
	}

	return h.showSettingsMenu(ctx, cb)
}

func (h *Handler) applyToggleTransliteration(ctx context.Context, cb *tgbotapi.CallbackQuery, _ string) error {
	if err := h.settingsService.ToggleTransliteration(ctx, cb.From.ID); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newHTMLMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}
		return err
	}

	confirm := tgbotapi.NewCallback(cb.ID, "Настройка транслитерации обновлена")
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send transliteration toggle confirmation",
			zap.Error(err),
		)
	}

	return h.showSettingsMenu(ctx, cb)
}

func (h *Handler) applyToggleAudio(ctx context.Context, cb *tgbotapi.CallbackQuery, _ string) error {
	if err := h.settingsService.ToggleAudio(ctx, cb.From.ID); err != nil {
		if errors.Is(err, repository.ErrSettingsNotFound) {
			msg := newHTMLMessage(cb.Message.Chat.ID, msgSettingsUnavailable)
			return h.send(msg)
		}
		return err
	}

	confirm := tgbotapi.NewCallback(cb.ID, "Настройка аудио обновлена")
	if _, err := h.bot.Request(confirm); err != nil {
		h.logger.Error("failed to send audio toggle confirmation",
			zap.Error(err),
		)
	}

	return h.showSettingsMenu(ctx, cb)
}
