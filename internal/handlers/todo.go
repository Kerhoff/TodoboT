package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
	"github.com/Kerhoff/TodoboT/internal/service"
)

// priorityEmoji returns an emoji representing the todo priority level.
func priorityEmoji(p models.TodoPriority) string {
	switch p {
	case models.TodoPriorityHigh:
		return "🔴"
	case models.TodoPriorityMedium:
		return "🟡"
	case models.TodoPriorityLow:
		return "🟢"
	default:
		return "⬜"
	}
}

// ---------------------------------------------------------------------------
// AddHandler – /add <text>
// ---------------------------------------------------------------------------

// AddHandler handles the /add command to create a new todo item.
type AddHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewAddHandler creates a new AddHandler.
func NewAddHandler(svc *service.Service, logger *logrus.Logger) *AddHandler {
	return &AddHandler{svc: svc, logger: logger}
}

// Handle processes the /add command.
func (h *AddHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a todo text.\nUsage: `/add Buy groceries`")
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
		chatTitle = message.From.FirstName + "'s list"
	}
	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, chatTitle)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}
	_ = h.svc.EnsureFamilyMember(ctx, family.ID, user.ID)

	title := strings.Join(args, " ")
	todo := &models.Todo{
		Title:       title,
		Status:      models.TodoStatusPending,
		Priority:    models.TodoPriorityMedium,
		CreatedByID: user.ID,
		ChatID:      message.Chat.ID,
	}

	todo, err = h.svc.Todos.Create(ctx, todo)
	if err != nil {
		return fmt.Errorf("create todo: %w", err)
	}

	text := fmt.Sprintf("✅ *Todo added!*\n\n🟡 *#%d* — %s", todo.ID, todo.Title)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"todo_id": todo.ID,
	}).Info("Todo created")

	return nil
}

// ---------------------------------------------------------------------------
// ListHandler – /list
// ---------------------------------------------------------------------------

// ListHandler handles the /list command to display pending todos for the chat.
type ListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewListHandler creates a new ListHandler.
func NewListHandler(svc *service.Service, logger *logrus.Logger) *ListHandler {
	return &ListHandler{svc: svc, logger: logger}
}

// Handle processes the /list command.
func (h *ListHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	status := models.TodoStatusPending
	filters := repository.TodoFilters{Status: &status}

	todos, err := h.svc.Todos.GetByChatID(ctx, message.Chat.ID, filters)
	if err != nil {
		return fmt.Errorf("list todos: %w", err)
	}

	if len(todos) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"📋 *No pending todos!*\n\nAdd one with `/add <text>`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("📋 *Pending Todos*\n\n")

	for i, t := range todos {
		sb.WriteString(fmt.Sprintf("%d. %s *#%d* %s", i+1, priorityEmoji(t.Priority), t.ID, t.Title))
		if t.Deadline != nil {
			sb.WriteString(fmt.Sprintf("  📅 _%s_", t.Deadline.Format("2006-01-02")))
		}
		if t.IsOverdue() {
			sb.WriteString(" ⚠️")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\n_%d pending items_", len(todos)))

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"count":   len(todos),
	}).Info("Listed todos")

	return nil
}

// ---------------------------------------------------------------------------
// DoneHandler – /done <id>
// ---------------------------------------------------------------------------

// DoneHandler handles the /done command to mark a todo as completed.
// It validates that the caller is the creator or assignee before allowing
// the status change.
type DoneHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewDoneHandler creates a new DoneHandler.
func NewDoneHandler(svc *service.Service, logger *logrus.Logger) *DoneHandler {
	return &DoneHandler{svc: svc, logger: logger}
}

// Handle processes the /done command.
func (h *DoneHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a todo ID.\nUsage: `/done 5`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	todoID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Invalid ID. Please provide a numeric todo ID.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	todo, err := h.svc.Todos.GetByID(ctx, todoID)
	if err != nil || todo == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Todo *#%d* not found.", todoID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Validate that the todo belongs to this chat
	if todo.ChatID != message.Chat.ID {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Todo *#%d* not found in this chat.", todoID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Validate ownership: only creator or assignee can mark as done
	isOwner := todo.CreatedByID == user.ID
	isAssignee := todo.AssignedToID != nil && *todo.AssignedToID == user.ID
	if !isOwner && !isAssignee {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ You can only complete todos you created or that are assigned to you.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if todo.IsCompleted() {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("ℹ️ Todo *#%d* is already completed.", todoID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	todo.Status = models.TodoStatusCompleted
	if _, err = h.svc.Todos.Update(ctx, todo); err != nil {
		return fmt.Errorf("complete todo: %w", err)
	}

	text := fmt.Sprintf("🎉 Todo *#%d* completed!\n\n~%s~", todo.ID, todo.Title)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"todo_id": todo.ID,
	}).Info("Todo completed")

	return nil
}

// ---------------------------------------------------------------------------
// DeleteHandler – /delete <id>
// ---------------------------------------------------------------------------

// DeleteHandler handles the /delete command to remove a todo.
// Only the creator of the todo is allowed to delete it.
type DeleteHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewDeleteHandler creates a new DeleteHandler.
func NewDeleteHandler(svc *service.Service, logger *logrus.Logger) *DeleteHandler {
	return &DeleteHandler{svc: svc, logger: logger}
}

// Handle processes the /delete command.
func (h *DeleteHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a todo ID.\nUsage: `/delete 5`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	todoID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Invalid ID. Please provide a numeric todo ID.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	todo, err := h.svc.Todos.GetByID(ctx, todoID)
	if err != nil || todo == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Todo *#%d* not found.", todoID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if todo.ChatID != message.Chat.ID {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Todo *#%d* not found in this chat.", todoID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Only the creator can delete a todo
	if todo.CreatedByID != user.ID {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ You can only delete todos you created.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if err = h.svc.Todos.Delete(ctx, todoID); err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}

	text := fmt.Sprintf("🗑 Todo *#%d* deleted: %s", todo.ID, todo.Title)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"todo_id": todo.ID,
	}).Info("Todo deleted")

	return nil
}

// ---------------------------------------------------------------------------
// MyHandler – /my
// ---------------------------------------------------------------------------

// MyHandler handles the /my command to show the current user's todos.
// It displays all pending todos the user created or that are assigned to them.
type MyHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewMyHandler creates a new MyHandler.
func NewMyHandler(svc *service.Service, logger *logrus.Logger) *MyHandler {
	return &MyHandler{svc: svc, logger: logger}
}

// Handle processes the /my command.
func (h *MyHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	// Fetch all pending todos for this chat, then filter for the user's own
	// items (created by the user or explicitly assigned to the user).
	status := models.TodoStatusPending
	filters := repository.TodoFilters{Status: &status}

	allTodos, err := h.svc.Todos.GetByChatID(ctx, message.Chat.ID, filters)
	if err != nil {
		return fmt.Errorf("get todos: %w", err)
	}

	var myTodos []*models.Todo
	for _, t := range allTodos {
		isCreator := t.CreatedByID == user.ID
		isAssignee := t.AssignedToID != nil && *t.AssignedToID == user.ID
		if isCreator || isAssignee {
			myTodos = append(myTodos, t)
		}
	}

	if len(myTodos) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"📌 *You have no pending todos!*\n\nCreate one with `/add <text>`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("📌 *Your Todos*\n\n")

	for i, t := range myTodos {
		sb.WriteString(fmt.Sprintf("%d. %s *#%d* %s", i+1, priorityEmoji(t.Priority), t.ID, t.Title))
		if t.Deadline != nil {
			sb.WriteString(fmt.Sprintf("  📅 _%s_", t.Deadline.Format("2006-01-02")))
		}
		if t.IsOverdue() {
			sb.WriteString(" ⚠️")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\n_%d items assigned to you_", len(myTodos)))

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"count":   len(myTodos),
	}).Info("Listed user's todos")

	return nil
}
