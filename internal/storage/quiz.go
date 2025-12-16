package storage

import (
	"sync"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type QuizStorage struct {
	mu        sync.RWMutex
	questions map[int64][]entities.Question
}

func NewQuizStorage() *QuizStorage {
	return &QuizStorage{
		questions: make(map[int64][]entities.Question),
	}
}

func (s *QuizStorage) Store(sessionID int64, questions []entities.Question) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.questions[sessionID] = questions
}

func (s *QuizStorage) Get(sessionID int64) []entities.Question {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.questions[sessionID]
}

func (s *QuizStorage) Delete(sessionID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.questions, sessionID)
}
