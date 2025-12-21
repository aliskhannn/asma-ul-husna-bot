package storage

import (
	"sync"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// QuizStorage provides in-memory storage for quiz questions by session ID.
type QuizStorage struct {
	mu        sync.RWMutex
	questions map[int64][]entities.Question
}

// NewQuizStorage creates a new QuizStorage.
func NewQuizStorage() *QuizStorage {
	return &QuizStorage{
		questions: make(map[int64][]entities.Question),
	}
}

// Store saves a list of questions for a given session ID.
func (s *QuizStorage) Store(sessionID int64, questions []entities.Question) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.questions[sessionID] = questions
}

// Get retrieves the list of questions for a given session ID.
func (s *QuizStorage) Get(sessionID int64) []entities.Question {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.questions[sessionID]
}

// Delete removes questions for a given session ID.
func (s *QuizStorage) Delete(sessionID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.questions, sessionID)
}
