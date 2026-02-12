package models

import "time"

// CalendarEvent represents a family calendar event
type CalendarEvent struct {
	ID          int64      `json:"id" db:"id"`
	FamilyID    int64      `json:"family_id" db:"family_id"`
	ChatID      int64      `json:"chat_id" db:"chat_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	StartTime   time.Time  `json:"start_time" db:"start_time"`
	EndTime     *time.Time `json:"end_time" db:"end_time"`
	AllDay      bool       `json:"all_day" db:"all_day"`
	Recurring   string     `json:"recurring" db:"recurring"` // none, daily, weekly, monthly, yearly
	Location    string     `json:"location" db:"location"`
	CreatedByID int64      `json:"created_by_id" db:"created_by_id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy   *User      `json:"created_by,omitempty"`
}

// IsUpcoming returns true if the event hasn't started yet
func (e *CalendarEvent) IsUpcoming() bool {
	return time.Now().Before(e.StartTime)
}

// IsOngoing returns true if the event is currently happening
func (e *CalendarEvent) IsOngoing() bool {
	now := time.Now()
	if e.EndTime == nil {
		return now.After(e.StartTime) && now.Before(e.StartTime.Add(time.Hour))
	}
	return now.After(e.StartTime) && now.Before(*e.EndTime)
}
