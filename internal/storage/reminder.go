package storage

import (
	"sync"
	"time"
)

type ReminderMessage struct {
	ChatID    int64
	MessageID int
	SentAt    time.Time
}

type ReminderStorage struct {
	mu       sync.RWMutex
	messages map[int64]ReminderMessage
}

func NewReminderStorage() *ReminderStorage {
	return &ReminderStorage{
		messages: make(map[int64]ReminderMessage),
	}
}

func (s *ReminderStorage) Store(userID int64, chatID int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages[userID] = ReminderMessage{
		ChatID:    chatID,
		MessageID: messageID,
		SentAt:    time.Now(),
	}
}

func (s *ReminderStorage) Get(userID int64) (ReminderMessage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg, ok := s.messages[userID]
	return msg, ok
}

func (s *ReminderStorage) Delete(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.messages, userID)
}

func (s *ReminderStorage) UpsertAndGetPrev(userID int64, chatID int64, messageID int) (prev ReminderMessage, hadPrev bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev, hadPrev = s.messages[userID]

	s.messages[userID] = ReminderMessage{
		ChatID:    chatID,
		MessageID: messageID,
		SentAt:    time.Now(),
	}

	return prev, hadPrev
}
