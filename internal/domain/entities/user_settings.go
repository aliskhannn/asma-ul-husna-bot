package entities

import "time"

type UserSettings struct {
	UserID              int64
	NamesPerDay         int
	QuizLength          int
	QuizMode            string
	ShowTransliteration bool
	ShowAudio           bool
	LanguageCode        *string // nullable, "ru", "en"
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
