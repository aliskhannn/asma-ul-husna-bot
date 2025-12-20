package entities

import "time"

// User represents bot user.
type User struct {
	ID        int64 // Telegram user ID
	ChatID    int64
	IsActive  bool
	CreatedAt time.Time
}

func NewUser(id, chatID int64) *User {
	return &User{
		ID:     id,
		ChatID: chatID,
	}
}
