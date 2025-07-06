package models

import "time"

// TodoStatus represents the status of a todo item
type TodoStatus string

const (
	TodoStatusPending   TodoStatus = "pending"
	TodoStatusCompleted TodoStatus = "completed"
	TodoStatusCancelled TodoStatus = "cancelled"
)

// TodoPriority represents the priority level of a todo item
type TodoPriority string

const (
	TodoPriorityLow    TodoPriority = "low"
	TodoPriorityMedium TodoPriority = "medium"
	TodoPriorityHigh   TodoPriority = "high"
)

// Todo represents a todo item
type Todo struct {
	ID           int64         `json:"id" db:"id"`
	Title        string        `json:"title" db:"title"`
	Description  string        `json:"description" db:"description"`
	Status       TodoStatus    `json:"status" db:"status"`
	Priority     TodoPriority  `json:"priority" db:"priority"`
	Deadline     *time.Time    `json:"deadline" db:"deadline"`
	CreatedByID  int64         `json:"created_by_id" db:"created_by_id"`
	AssignedToID *int64        `json:"assigned_to_id" db:"assigned_to_id"`
	ChatID       int64         `json:"chat_id" db:"chat_id"`
	MessageID    *int64        `json:"message_id" db:"message_id"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
	CreatedBy    *User         `json:"created_by,omitempty"`
	AssignedTo   *User         `json:"assigned_to,omitempty"`
	Comments     []Comment     `json:"comments,omitempty"`
}

// IsCompleted returns true if the todo is completed
func (t *Todo) IsCompleted() bool {
	return t.Status == TodoStatusCompleted
}

// IsPending returns true if the todo is pending
func (t *Todo) IsPending() bool {
	return t.Status == TodoStatusPending
}

// IsAssigned returns true if the todo is assigned to someone
func (t *Todo) IsAssigned() bool {
	return t.AssignedToID != nil
}

// IsOverdue returns true if the todo has a deadline and it's passed
func (t *Todo) IsOverdue() bool {
	if t.Deadline == nil || t.IsCompleted() {
		return false
	}
	return time.Now().After(*t.Deadline)
}