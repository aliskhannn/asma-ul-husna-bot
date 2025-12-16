package entities

import "time"

// User represents bot user.
type User struct {
	ID        int64 // Telegram user ID
	IsActive  bool
	CreatedAt time.Time
}

func NewUser(id int64, firstName string, lastName, username, language *string) *User {
	return &User{
		ID: id,
	}
}
