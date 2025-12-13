package entities

import "time"

type User struct {
	ID           int64
	FirstName    string
	LastName     string
	Username     string
	LanguageCode string
	IsActive     bool
	CreatedAt    time.Time
}

func NewUser(
	ID int64,
	FirstName string,
	LastName string,
	Username string,
	Language string,
	IsActive bool,
) *User {
	return &User{
		ID:           ID,
		FirstName:    FirstName,
		LastName:     LastName,
		Username:     Username,
		LanguageCode: Language,
		IsActive:     IsActive,
		CreatedAt:    time.Now(),
	}
}
