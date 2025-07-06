package models

import "time"

// User represents a Telegram user in the system
type User struct {
	ID               int64     `json:"id" db:"id"`
	TelegramID       int64     `json:"telegram_id" db:"telegram_id"`
	TelegramUsername string    `json:"telegram_username" db:"telegram_username"`
	FirstName        string    `json:"first_name" db:"first_name"`
	LastName         string    `json:"last_name" db:"last_name"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// FullName returns the user's full name
func (u *User) FullName() string {
	if u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	return u.FirstName
}

// DisplayName returns the best display name for the user
func (u *User) DisplayName() string {
	if u.TelegramUsername != "" {
		return "@" + u.TelegramUsername
	}
	return u.FullName()
}