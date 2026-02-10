package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/service"
)

var (
	relativeTimeRegex = regexp.MustCompile(`^(\d+)([mhd])$`)
	clockTimeRegex    = regexp.MustCompile(`^\d{1,2}:\d{2}$`)
	dateOnlyRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// parseRemindTime parses the time specification from the beginning of args.
// It returns the parsed reminder time, the index of the first text arg, and
// any error.
//
// Supported formats:
//   - "10m", "2h", "1d"             -> relative time from now
//   - "15:30"                        -> today at that time (tomorrow if passed)
//   - "2025-12-31 15:30"            -> absolute datetime (consumes two args)
func parseRemindTime(args []string) (time.Time, int, error) {
	if len(args) == 0 {
		return time.Time{}, 0, fmt.Errorf("no time specified")
	}

	now := time.Now()

	// 1) Relative time: 10m, 2h, 1d
	if matches := relativeTimeRegex.FindStringSubmatch(args[0]); matches != nil {
		value, _ := strconv.Atoi(matches[1])
		var d time.Duration
		switch matches[2] {
		case "m":
			d = time.Duration(value) * time.Minute
		case "h":
			d = time.Duration(value) * time.Hour
		case "d":
			d = time.Duration(value) * 24 * time.Hour
		}
		return now.Add(d), 1, nil
	}

	// 2) Absolute datetime: 2025-12-31 15:30 (two args)
	if len(args) >= 2 && dateOnlyRegex.MatchString(args[0]) && clockTimeRegex.MatchString(args[1]) {
		t, err := time.ParseInLocation("2006-01-02 15:04", args[0]+" "+args[1], time.Local)
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("invalid datetime: %s %s", args[0], args[1])
		}
		return t, 2, nil
	}

	// 3) Clock time only: 15:30
	if clockTimeRegex.MatchString(args[0]) {
		t, err := time.Parse("15:04", args[0])
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("invalid time: %s", args[0])
		}
		remindAt := time.Date(now.Year(), now.Month(), now.Day(),
			t.Hour(), t.Minute(), 0, 0, time.Local)
		// If the time has already passed today, schedule for tomorrow
		if remindAt.Before(now) {
			remindAt = remindAt.AddDate(0, 0, 1)
		}
		return remindAt, 1, nil
	}

	return time.Time{}, 0, fmt.Errorf("unrecognized time format: %s", args[0])
}

// formatReminderTime produces a human-readable string for when a reminder
// is scheduled to fire. For times less than 24 h away it shows a relative
// duration together with the clock time; otherwise it shows the full date.
func formatReminderTime(t time.Time) string {
	now := time.Now()
	diff := t.Sub(now)

	if diff < 0 {
		return t.Format("2006-01-02 15:04") + " (overdue)"
	}

	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		minutes := int(diff.Minutes()) % 60
		if hours > 0 {
			return fmt.Sprintf("in %dh %dm (%s)", hours, minutes, t.Format("15:04"))
		}
		if minutes > 0 {
			return fmt.Sprintf("in %dm (%s)", minutes, t.Format("15:04"))
		}
		return fmt.Sprintf("in less than a minute (%s)", t.Format("15:04"))
	}

	return t.Format("Mon, 02 Jan 2006 at 15:04")
}

// ---------------------------------------------------------------------------
// RemindHandler – /remind <time> <text>
// ---------------------------------------------------------------------------

// RemindHandler handles the /remind command to create a reminder.
type RemindHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewRemindHandler creates a new RemindHandler.
func NewRemindHandler(svc *service.Service, logger *logrus.Logger) *RemindHandler {
	return &RemindHandler{svc: svc, logger: logger}
}

// Handle processes the /remind command.
func (h *RemindHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) < 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a time and reminder text.\n\n"+
				"*Usage:*\n"+
				"`/remind 10m Take out trash`\n"+
				"`/remind 2h Call dentist`\n"+
				"`/remind 1d Pay bills`\n"+
				"`/remind 15:30 Pick up kids`\n"+
				"`/remind 2025-12-31 15:30 New Year party`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	remindAt, textStart, err := parseRemindTime(args)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Could not parse time.\n\n"+
				"Supported formats: `10m`, `2h`, `1d`, `15:30`, `2025-12-31 15:30`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if textStart >= len(args) {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a reminder text after the time.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	reminderText := strings.Join(args[textStart:], " ")

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	chatTitle := message.Chat.Title
	if chatTitle == "" {
		chatTitle = message.From.FirstName + "'s reminders"
	}
	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, chatTitle)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}
	_ = h.svc.EnsureFamilyMember(ctx, family.ID, user.ID)

	reminder := &models.Reminder{
		FamilyID: family.ID,
		ChatID:   message.Chat.ID,
		UserID:   user.ID,
		Text:     reminderText,
		RemindAt: remindAt,
		Repeat:   models.ReminderRepeatNone,
		Active:   true,
	}

	reminder, err = h.svc.Reminders.Create(ctx, reminder)
	if err != nil {
		return fmt.Errorf("create reminder: %w", err)
	}

	text := fmt.Sprintf("⏰ *Reminder set!*\n\n*#%d* — %s\n📅 %s",
		reminder.ID, reminderText, formatReminderTime(remindAt))
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id":     message.Chat.ID,
		"user_id":     message.From.ID,
		"reminder_id": reminder.ID,
		"remind_at":   remindAt,
	}).Info("Reminder created")

	return nil
}

// ---------------------------------------------------------------------------
// RemindersListHandler – /reminders
// ---------------------------------------------------------------------------

// RemindersListHandler handles the /reminders command to list the current
// user's active reminders.
type RemindersListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewRemindersListHandler creates a new RemindersListHandler.
func NewRemindersListHandler(svc *service.Service, logger *logrus.Logger) *RemindersListHandler {
	return &RemindersListHandler{svc: svc, logger: logger}
}

// Handle processes the /reminders command.
func (h *RemindersListHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	reminders, err := h.svc.Reminders.GetByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("list reminders: %w", err)
	}

	// Keep only active reminders
	var active []*models.Reminder
	for _, r := range reminders {
		if r.Active {
			active = append(active, r)
		}
	}

	if len(active) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⏰ *No active reminders!*\n\nCreate one with `/remind <time> <text>`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("⏰ *Your Reminders*\n\n")

	for i, r := range active {
		sb.WriteString(fmt.Sprintf("%d. *#%d* %s\n   📅 %s", i+1, r.ID, r.Text, formatReminderTime(r.RemindAt)))
		if r.Repeat != models.ReminderRepeatNone {
			sb.WriteString(fmt.Sprintf(" (🔁 %s)", string(r.Repeat)))
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("_%d active reminders_\n\n_Delete with_ `/delremind <id>`", len(active)))

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"count":   len(active),
	}).Info("Listed reminders")

	return nil
}

// ---------------------------------------------------------------------------
// RemindDeleteHandler – /delremind <id>
// ---------------------------------------------------------------------------

// RemindDeleteHandler handles the /delremind command to delete a reminder.
// Only the owner of the reminder is allowed to delete it.
type RemindDeleteHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewRemindDeleteHandler creates a new RemindDeleteHandler.
func NewRemindDeleteHandler(svc *service.Service, logger *logrus.Logger) *RemindDeleteHandler {
	return &RemindDeleteHandler{svc: svc, logger: logger}
}

// Handle processes the /delremind command.
func (h *RemindDeleteHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a reminder ID.\nUsage: `/delremind 3`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	reminderID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Invalid ID. Please provide a numeric reminder ID.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	reminder, err := h.svc.Reminders.GetByID(ctx, reminderID)
	if err != nil || reminder == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Reminder *#%d* not found.", reminderID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Only the owner can delete their reminder
	if reminder.UserID != user.ID {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ You can only delete your own reminders.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if err = h.svc.Reminders.Delete(ctx, reminderID); err != nil {
		return fmt.Errorf("delete reminder: %w", err)
	}

	text := fmt.Sprintf("🗑 Reminder *#%d* deleted: %s", reminder.ID, reminder.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id":     message.Chat.ID,
		"user_id":     message.From.ID,
		"reminder_id": reminder.ID,
	}).Info("Reminder deleted")

	return nil
}
