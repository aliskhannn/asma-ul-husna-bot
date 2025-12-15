package telegram

import (
	"fmt"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

const lrm = "\u200E"

func processName(n entities.Name) string {
	return fmt.Sprintf(
		"%s<b>%d. </b>%s\n\n<b>Транслитерация:</b>  %s\n<b>Перевод:</b> %s\n<b>Значение:</b> %s",
		lrm,
		n.Number,
		n.ArabicName,
		n.Transliteration,
		n.Translation,
		n.Meaning,
	)
}

func buildNameAudio(name entities.Name, chatID int64) *tgbotapi.AudioConfig {
	path := filepath.Join("assets", "audio", name.Audio)

	a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(path))
	a.Caption = name.Transliteration

	return &a
}

func newHTMLMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	return msg
}
