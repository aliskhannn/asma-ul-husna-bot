package entities

import "time"

// User represents a bot user.
type User struct {
	ID        int64     // Telegram user ID
	ChatID    int64     // Telegram chat ID
	IsActive  bool      // whether the user is active
	CreatedAt time.Time // timestamp when the user was created
}

// NewUser creates a new user with the specified Telegram ID and chat ID.
func NewUser(id, chatID int64) *User {
	return &User{
		ID:        id,
		ChatID:    chatID,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
}
