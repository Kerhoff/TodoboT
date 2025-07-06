package models

import "time"

// Comment represents a comment on a todo item
type Comment struct {
	ID        int64     `json:"id" db:"id"`
	TodoID    int64     `json:"todo_id" db:"todo_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	User      *User     `json:"user,omitempty"`
}