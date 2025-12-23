package storage

import (
	"sync"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// QuizStorage provides in-memory storage for quiz questions by session ID.
type QuizStorage struct {
	mu         sync.RWMutex
	names      map[int64][]entities.Name
	messageIDs map[int64]int
}

// NewQuizStorage creates a new QuizStorage.
func NewQuizStorage() *QuizStorage {
	return &QuizStorage{
		names:      make(map[int64][]entities.Name),
		messageIDs: make(map[int64]int),
	}
}

// Store saves a list of questions for a given session ID.
func (s *QuizStorage) Store(sessionID int64, names []entities.Name) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.names[sessionID] = names
}

// Get retrieves the list of questions for a given session ID.
func (s *QuizStorage) Get(sessionID int64) []entities.Name {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.names[sessionID]
}

// Delete removes questions for a given session ID.
func (s *QuizStorage) Delete(sessionID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.names, sessionID)
	delete(s.messageIDs, sessionID)
}

// StoreMessageID saves the message ID of the last quiz question.
func (s *QuizStorage) StoreMessageID(sessionID int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageIDs[sessionID] = messageID
}

// GetMessageID retrieves the message ID of the last quiz question.
func (s *QuizStorage) GetMessageID(sessionID int64) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgID, exists := s.messageIDs[sessionID]
	return msgID, exists
}

// DeleteMessageID removes the stored message ID.
func (s *QuizStorage) DeleteMessageID(sessionID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.messageIDs, sessionID)
}
