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
	"github.com/Kerhoff/TodoboT/internal/repository"
	"github.com/Kerhoff/TodoboT/internal/service"
)

var (
	calDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	calTimeRegex = regexp.MustCompile(`^\d{1,2}:\d{2}$`)
)

// ---------------------------------------------------------------------------
// CalendarAddHandler – /event <title> <date> [time]
// ---------------------------------------------------------------------------

// CalendarAddHandler handles the /event command to create a calendar event.
// It parses the date (YYYY-MM-DD) and optional time (HH:MM) from the end of
// the argument list; everything before is treated as the event title.
type CalendarAddHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewCalendarAddHandler creates a new CalendarAddHandler.
func NewCalendarAddHandler(svc *service.Service, logger *logrus.Logger) *CalendarAddHandler {
	return &CalendarAddHandler{svc: svc, logger: logger}
}

// Handle processes the /event command.
func (h *CalendarAddHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) < 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a title and date.\n\n"+
				"*Usage:*\n"+
				"`/event Meeting 2025-01-15 14:00`\n"+
				"`/event Birthday party 2025-03-20`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Parse from the end: optional time, then date, rest is the title.
	var dateStr, timeStr string
	lastIdx := len(args) - 1

	// Check if last arg is a time (HH:MM)
	if calTimeRegex.MatchString(args[lastIdx]) {
		timeStr = args[lastIdx]
		lastIdx--
	}

	// Check if current last arg is a date (YYYY-MM-DD)
	if lastIdx >= 0 && calDateRegex.MatchString(args[lastIdx]) {
		dateStr = args[lastIdx]
		lastIdx--
	}

	if dateStr == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Could not find a date in your command.\n"+
				"Please use the format `YYYY-MM-DD`.\n"+
				"Example: `/event Meeting 2025-01-15 14:00`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	titleParts := args[:lastIdx+1]
	if len(titleParts) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide an event title before the date.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}
	title := strings.Join(titleParts, " ")

	// Parse date and optional time
	var startTime time.Time
	var allDay bool
	var parseErr error

	if timeStr != "" {
		startTime, parseErr = time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, time.Local)
		allDay = false
	} else {
		startTime, parseErr = time.ParseInLocation("2006-01-02", dateStr, time.Local)
		allDay = true
	}
	if parseErr != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Invalid date/time format.\nDate: `YYYY-MM-DD`, Time: `HH:MM`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	chatTitle := message.Chat.Title
	if chatTitle == "" {
		chatTitle = message.From.FirstName + "'s calendar"
	}
	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, chatTitle)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}
	_ = h.svc.EnsureFamilyMember(ctx, family.ID, user.ID)

	event := &models.CalendarEvent{
		FamilyID:    family.ID,
		ChatID:      message.Chat.ID,
		Title:       title,
		StartTime:   startTime,
		AllDay:      allDay,
		Recurring:   "none",
		CreatedByID: user.ID,
	}

	event, err = h.svc.Calendar.Create(ctx, event)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	var dateDisplay string
	if allDay {
		dateDisplay = startTime.Format("Mon, 02 Jan 2006") + " (all day)"
	} else {
		dateDisplay = startTime.Format("Mon, 02 Jan 2006 at 15:04")
	}

	text := fmt.Sprintf("📅 *Event created!*\n\n*#%d* — %s\n📆 %s", event.ID, title, dateDisplay)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id":  message.Chat.ID,
		"user_id":  message.From.ID,
		"event_id": event.ID,
	}).Info("Calendar event created")

	return nil
}

// ---------------------------------------------------------------------------
// CalendarListHandler – /events
// ---------------------------------------------------------------------------

// CalendarListHandler handles the /events command to list upcoming events
// for the current chat.
type CalendarListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewCalendarListHandler creates a new CalendarListHandler.
func NewCalendarListHandler(svc *service.Service, logger *logrus.Logger) *CalendarListHandler {
	return &CalendarListHandler{svc: svc, logger: logger}
}

// Handle processes the /events command.
func (h *CalendarListHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	now := time.Now().Format("2006-01-02 15:04:05")
	filters := repository.CalendarFilters{
		From:  &now,
		Limit: 20,
	}

	events, err := h.svc.Calendar.GetByChatID(ctx, message.Chat.ID, filters)
	if err != nil {
		return fmt.Errorf("list events: %w", err)
	}

	if len(events) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"📅 *No upcoming events!*\n\nAdd one with `/event <title> <date> [time]`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("📅 *Upcoming Events*\n\n")

	for i, event := range events {
		var dateDisplay string
		if event.AllDay {
			dateDisplay = event.StartTime.Format("Mon, 02 Jan 2006") + " (all day)"
		} else {
			dateDisplay = event.StartTime.Format("Mon, 02 Jan 2006 at 15:04")
		}

		status := "📆"
		if event.IsOngoing() {
			status = "▶️"
		}

		sb.WriteString(fmt.Sprintf("%d. %s *#%d* %s\n   📆 %s", i+1, status, event.ID, event.Title, dateDisplay))
		if event.Location != "" {
			sb.WriteString(fmt.Sprintf("\n   📍 %s", event.Location))
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("_%d upcoming events_", len(events)))

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"count":   len(events),
	}).Info("Listed calendar events")

	return nil
}

// ---------------------------------------------------------------------------
// CalendarDeleteHandler – /delevent <id>
// ---------------------------------------------------------------------------

// CalendarDeleteHandler handles the /delevent command to delete a calendar event.
// Only the creator of the event is allowed to delete it.
type CalendarDeleteHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewCalendarDeleteHandler creates a new CalendarDeleteHandler.
func NewCalendarDeleteHandler(svc *service.Service, logger *logrus.Logger) *CalendarDeleteHandler {
	return &CalendarDeleteHandler{svc: svc, logger: logger}
}

// Handle processes the /delevent command.
func (h *CalendarDeleteHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide an event ID.\nUsage: `/delevent 3`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	eventID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Invalid ID. Please provide a numeric event ID.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	event, err := h.svc.Calendar.GetByID(ctx, eventID)
	if err != nil || event == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Event *#%d* not found.", eventID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if event.ChatID != message.Chat.ID {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Event *#%d* not found in this chat.", eventID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Only the creator can delete the event
	if event.CreatedByID != user.ID {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ You can only delete events you created.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if err = h.svc.Calendar.Delete(ctx, eventID); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	text := fmt.Sprintf("🗑 Event *#%d* deleted: %s", event.ID, event.Title)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id":  message.Chat.ID,
		"user_id":  message.From.ID,
		"event_id": event.ID,
	}).Info("Calendar event deleted")

	return nil
}
