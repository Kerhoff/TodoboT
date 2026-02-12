package models

import "time"

// ReminderRepeat defines how often a reminder repeats
type ReminderRepeat string

const (
	ReminderRepeatNone    ReminderRepeat = "none"
	ReminderRepeatDaily   ReminderRepeat = "daily"
	ReminderRepeatWeekly  ReminderRepeat = "weekly"
	ReminderRepeatMonthly ReminderRepeat = "monthly"
)

// Reminder represents a scheduled reminder
type Reminder struct {
	ID          int64          `json:"id" db:"id"`
	FamilyID    int64          `json:"family_id" db:"family_id"`
	ChatID      int64          `json:"chat_id" db:"chat_id"`
	UserID      int64          `json:"user_id" db:"user_id"`
	Text        string         `json:"text" db:"text"`
	RemindAt    time.Time      `json:"remind_at" db:"remind_at"`
	Repeat      ReminderRepeat `json:"repeat" db:"repeat_interval"`
	Active      bool           `json:"active" db:"active"`
	LastSentAt  *time.Time     `json:"last_sent_at" db:"last_sent_at"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
	User        *User          `json:"user,omitempty"`
}

// IsDue returns true if the reminder should fire now
func (r *Reminder) IsDue() bool {
	if !r.Active {
		return false
	}
	return time.Now().After(r.RemindAt)
}

// NextRemindAt calculates the next reminder time based on repeat interval
func (r *Reminder) NextRemindAt() time.Time {
	switch r.Repeat {
	case ReminderRepeatDaily:
		return r.RemindAt.AddDate(0, 0, 1)
	case ReminderRepeatWeekly:
		return r.RemindAt.AddDate(0, 0, 7)
	case ReminderRepeatMonthly:
		return r.RemindAt.AddDate(0, 1, 0)
	default:
		return r.RemindAt
	}
}
