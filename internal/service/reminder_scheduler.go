package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
)

// ReminderCallback is a function that sends a reminder message to a chat.
type ReminderCallback func(chatID int64, text string)

// StartReminderScheduler runs a background loop that checks for due reminders
// every 30 seconds and invokes the callback for each one. It blocks until the
// context is cancelled, so it should be launched in a separate goroutine.
func (s *Service) StartReminderScheduler(ctx context.Context, callback ReminderCallback) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.logger.Info("Reminder scheduler started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Reminder scheduler stopped")
			return
		case <-ticker.C:
			s.processReminders(ctx, callback)
		}
	}
}

// processReminders fetches all due reminders and fires the callback for each
// one. After sending, it either deactivates one-time reminders or advances
// repeating reminders to their next scheduled time.
func (s *Service) processReminders(ctx context.Context, callback ReminderCallback) {
	reminders, err := s.Reminders.GetDue(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get due reminders: %v", err)
		return
	}

	for _, r := range reminders {
		callback(r.ChatID, fmt.Sprintf("\u23f0 *Reminder*\n%s", r.Text))

		now := time.Now()
		r.LastSentAt = &now

		if r.Repeat == models.ReminderRepeatNone {
			r.Active = false
		} else {
			r.RemindAt = r.NextRemindAt()
		}
		r.UpdatedAt = now

		if _, err := s.Reminders.Update(ctx, r); err != nil {
			s.logger.Errorf("Failed to update reminder %d: %v", r.ID, err)
		}
	}
}
