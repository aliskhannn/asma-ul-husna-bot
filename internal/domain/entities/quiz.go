package entities

import "time"

type QuizSession struct {
	ID                 int64
	UserID             int64
	CurrentQuestionNum int
	CorrectAnswers     int
	TotalQuestions     int
	QuizMode           string // "daily", "review", "custom"
	SessionStatus      string // "active", "completed", "abandoned"
	StartedAt          time.Time
	CompletedAt        *time.Time // nullable
}

type QuizAnswer struct {
	ID            int64
	UserID        int64
	SessionID     int64
	NameNumber    int
	UserAnswer    string
	CorrectAnswer string
	QuestionType  string // "translation", "transliteration", "meaning", "arabic"
	IsCorrect     bool
	AnsweredAt    time.Time
}
