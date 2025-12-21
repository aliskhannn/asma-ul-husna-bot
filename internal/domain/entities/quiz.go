package entities

import (
	"strings"
	"time"
)

// QuizSession represents a single quiz session for a user.
// It tracks the session ID, user ID, progress, quiz mode, session status, and timestamps.
type QuizSession struct {
	ID                 int64      // unique session ID
	UserID             int64      // user ID who started the quiz
	CurrentQuestionNum int        // current question number in the quiz
	CorrectAnswers     int        // number of correct answers so far
	TotalQuestions     int        // total number of questions in the quiz
	QuizMode           string     // quiz mode: "new", "review", or "mixed"
	SessionStatus      string     // session status: "active", "completed", or "abandoned"
	StartedAt          time.Time  // timestamp when the quiz started
	CompletedAt        *time.Time // timestamp when the quiz was completed (nullable)
}

// NewQuizSession creates a new quiz session for a user with the specified total questions and mode.
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

// Complete marks the quiz session as completed and sets the completion timestamp.
func (qs *QuizSession) Complete() {
	qs.SessionStatus = "completed"
	now := time.Now()
	qs.CompletedAt = &now
}

// QuizAnswer represents a user's answer to a quiz question.
// It tracks the answer details, correctness, and timestamp.
type QuizAnswer struct {
	ID            int64     // unique answer ID
	UserID        int64     // user ID who answered
	SessionID     int64     // quiz session ID
	NameNumber    int       // number of the associated name
	UserAnswer    string    // user's answer
	CorrectAnswer string    // correct answer
	QuestionType  string    // type of question: "translation", "transliteration", "meaning", or "arabic"
	IsCorrect     bool      // whether the answer was correct
	AnsweredAt    time.Time // timestamp when the answer was submitted
}

// NewQuizAnswer creates a new quiz answer for a user, session, and name.
func NewQuizAnswer(userID, sessionID int64, nameNumber int, questionType string) *QuizAnswer {
	return &QuizAnswer{
		UserID:       userID,
		SessionID:    sessionID,
		NameNumber:   nameNumber,
		QuestionType: questionType,
		AnsweredAt:   time.Now(),
	}
}

// CheckAnswer sets the user's answer, correct answer, and determines if the answer is correct.
func (qa *QuizAnswer) CheckAnswer(userAnswer, correctAnswer string) {
	qa.UserAnswer = userAnswer
	qa.CorrectAnswer = correctAnswer
	qa.IsCorrect = strings.EqualFold(
		strings.TrimSpace(userAnswer),
		strings.TrimSpace(correctAnswer),
	)
}
