package models

import "time"

// Family represents a family group (typically mapped to a Telegram group chat)
type Family struct {
	ID        int64     `json:"id" db:"id"`
	ChatID    int64     `json:"chat_id" db:"chat_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Members   []User    `json:"members,omitempty"`
}

// FamilyMember represents the join table between families and users
type FamilyMember struct {
	ID       int64  `json:"id" db:"id"`
	FamilyID int64  `json:"family_id" db:"family_id"`
	UserID   int64  `json:"user_id" db:"user_id"`
	Role     string `json:"role" db:"role"` // "admin" or "member"
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}
