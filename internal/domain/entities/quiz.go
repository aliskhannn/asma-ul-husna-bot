package entities

import (
	"strings"
	"time"
)

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

func NewQuizSession(userID int64, totalQuestions int, mode string) *QuizSession {
	return &QuizSession{
		UserID:             userID,
		CurrentQuestionNum: 1,
		CorrectAnswers:     0,
		TotalQuestions:     totalQuestions,
		QuizMode:           mode,
		SessionStatus:      "active",
		StartedAt:          time.Now(),
	}
}

func (qs *QuizSession) Complete() {
	qs.SessionStatus = "completed"
	now := time.Now()
	qs.CompletedAt = &now
}

func (qs *QuizSession) Abandon() {
	qs.SessionStatus = "abandoned"
	now := time.Now()
	qs.CompletedAt = &now
}

func (qs *QuizSession) Progress() float64 {
	if qs.TotalQuestions == 0 {
		return 0
	}
	return float64(qs.CurrentQuestionNum) / float64(qs.TotalQuestions) * 100
}

func (qs *QuizSession) Accuracy() float64 {
	if qs.CurrentQuestionNum == 0 {
		return 0
	}
	return float64(qs.CorrectAnswers) / float64(qs.CurrentQuestionNum) * 100
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

func NewQuizAnswer(userID, sessionID int64, nameNumber int, questionType string) *QuizAnswer {
	return &QuizAnswer{
		UserID:       userID,
		SessionID:    sessionID,
		NameNumber:   nameNumber,
		QuestionType: questionType,
		AnsweredAt:   time.Now(),
	}
}

func (qa *QuizAnswer) CheckAnswer(userAnswer, correctAnswer string) {
	qa.UserAnswer = userAnswer
	qa.CorrectAnswer = correctAnswer

	qa.IsCorrect = strings.EqualFold(
		strings.TrimSpace(userAnswer),
		strings.TrimSpace(correctAnswer),
	)
}
