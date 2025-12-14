package entities

import "time"

type User struct {
	ID           int64
	FirstName    string
	LastName     *string // nullable
	Username     *string // nullable
	LanguageCode *string // nullable
	IsActive     bool
	CreatedAt    time.Time
}

func NewUser(id int64, firstName string, lastName, username, language *string) *User {
	return &User{
		ID:           id,
		FirstName:    firstName,
		LastName:     lastName,
		Username:     username,
		LanguageCode: language,
	}
}
