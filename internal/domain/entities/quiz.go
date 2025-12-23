package entities

import (
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
	Version            int        // for optimistic locking
}

// QuizQuestion represents a single question in a quiz session.
type QuizQuestion struct {
	ID            int64
	SessionID     int64
	QuestionOrder int
	NameNumber    int
	QuestionType  string
	CorrectAnswer string
	Options       []string
	CorrectIndex  int
	CreatedAt     time.Time
}

// QuizAnswer represents a user's answer to a quiz question.
// It tracks the answer details, correctness, and timestamp.
type QuizAnswer struct {
	ID            int64 // unique answer ID
	UserID        int64 // user ID who answered
	SessionID     int64 // quiz session ID
	QuestionID    int64
	NameNumber    int       // number of the associated name
	UserAnswer    string    // user's answer
	CorrectAnswer string    // correct answer
	QuestionType  string    // type of question: "translation", "transliteration", "meaning", or "arabic"
	IsCorrect     bool      // whether the answer was correct
	AnsweredAt    time.Time // timestamp when the answer was submitted
}

// QuestionType represents the type of quiz question.
type QuestionType string

const (
	QuestionTypeTranslation     QuestionType = "translation"
	QuestionTypeTransliteration QuestionType = "transliteration"
	QuestionTypeMeaning         QuestionType = "meaning"
	QuestionTypeArabic          QuestionType = "arabic"
)

// IsActive returns true if the session is currently active.
func (q *QuizSession) IsActive() bool {
	return q.SessionStatus == "active"
}

// IsCompleted returns true if the session is completed.
func (s *QuizSession) IsCompleted() bool {
	return s.SessionStatus == "completed"
}

// MarkCompleted marks the session as completed.
func (s *QuizSession) MarkCompleted(now time.Time) {
	s.SessionStatus = "completed"
	s.CompletedAt = &now
}

// IncrementQuestion moves to the next question.
func (s *QuizSession) IncrementQuestion() {
	s.CurrentQuestionNum++
}

// IncrementCorrectAnswers increments the correct answers counter.
func (s *QuizSession) IncrementCorrectAnswers() {
	s.CorrectAnswers++
}

// ShouldComplete returns true if all questions have been answered.
func (s *QuizSession) ShouldComplete() bool {
	return s.CurrentQuestionNum > s.TotalQuestions
}

// AccuracyPercentage returns the accuracy as a percentage.
func (s *QuizSession) AccuracyPercentage() float64 {
	if s.TotalQuestions == 0 {
		return 0
	}
	return float64(s.CorrectAnswers) / float64(s.TotalQuestions) * 100
}
