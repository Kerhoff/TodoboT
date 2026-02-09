package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/service"
)

// RemindHandler handles the /remind command
type RemindHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewRemindHandler(svc *service.Service, logger *logrus.Logger) *RemindHandler {
	return &RemindHandler{svc: svc, logger: logger}
}

func (h *RemindHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) < 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Usage: /remind <time> <text>\nTime formats: 10m, 2h, 1d, 15:30, 2025-01-15 15:30")
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()
	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, message.Chat.Title)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}

	// Parse remind time
	remindAt, textStart, err := parseRemindTime(args)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Could not parse time. Formats: 10m, 2h, 1d, 15:30, 2025-01-15 15:30")
		bot.Send(msg)
		return nil
	}

	reminderText := strings.Join(args[textStart:], " ")
	if reminderText == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Please provide reminder text")
		bot.Send(msg)
		return nil
	}

	reminder := &models.Reminder{
		FamilyID: family.ID,
		ChatID:   message.Chat.ID,
		UserID:   user.ID,
		Text:     reminderText,
		RemindAt: remindAt,
		Repeat:   models.ReminderRepeatNone,
	}

	reminder, err = h.svc.Reminders.Create(ctx, reminder)
	if err != nil {
		return fmt.Errorf("create reminder: %w", err)
	}

	text := fmt.Sprintf("⏰ Reminder #%d set for *%s*\n📝 %s",
		reminder.ID, reminder.RemindAt.Format("Mon, 02 Jan 2006 15:04"), reminder.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

func parseRemindTime(args []string) (time.Time, int, error) {
	now := time.Now()

	// Try relative time: 10m, 2h, 1d
	if len(args[0]) >= 2 {
		numStr := args[0][:len(args[0])-1]
		unit := args[0][len(args[0])-1:]
		if num, err := strconv.Atoi(numStr); err == nil {
			switch unit {
			case "m":
				return now.Add(time.Duration(num) * time.Minute), 1, nil
			case "h":
				return now.Add(time.Duration(num) * time.Hour), 1, nil
			case "d":
				return now.AddDate(0, 0, num), 1, nil
			}
		}
	}

	// Try absolute date+time: 2025-01-15 15:30
	if len(args) >= 2 {
		if t, err := time.ParseInLocation("2006-01-02 15:04", args[0]+" "+args[1], time.Local); err == nil {
			return t, 2, nil
		}
	}

	// Try time only: 15:30 (today or tomorrow)
	if t, err := time.Parse("15:04", args[0]); err == nil {
		remindAt := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		if remindAt.Before(now) {
			remindAt = remindAt.AddDate(0, 0, 1)
		}
		return remindAt, 1, nil
	}

	return time.Time{}, 0, fmt.Errorf("could not parse time")
}

// RemindersListHandler handles the /reminders command
type RemindersListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewRemindersListHandler(svc *service.Service, logger *logrus.Logger) *RemindersListHandler {
	return &RemindersListHandler{svc: svc, logger: logger}
}

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

	if len(reminders) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "⏰ No active reminders. Set one with /remind")
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("⏰ *Your Reminders:*\n\n")
	for _, r := range reminders {
		repeat := ""
		if r.Repeat != models.ReminderRepeatNone {
			repeat = fmt.Sprintf(" (🔄 %s)", r.Repeat)
		}
		sb.WriteString(fmt.Sprintf("#%d: %s\n   📆 %s%s\n",
			r.ID, r.Text, r.RemindAt.Format("Mon, 02 Jan 15:04"), repeat))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// RemindDeleteHandler handles the /delremind command
type RemindDeleteHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewRemindDeleteHandler(svc *service.Service, logger *logrus.Logger) *RemindDeleteHandler {
	return &RemindDeleteHandler{svc: svc, logger: logger}
}

func (h *RemindDeleteHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Usage: /delremind <id>")
		bot.Send(msg)
		return nil
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Invalid reminder ID")
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()
	if err := h.svc.Reminders.Delete(ctx, id); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Reminder not found")
		bot.Send(msg)
		return nil
	}

	text := fmt.Sprintf("🗑 Reminder #%d deleted", id)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	bot.Send(msg)
	return nil
}
