package handlers

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// StartHandler handles the /start command
type StartHandler struct {
	logger *logrus.Logger
}

// NewStartHandler creates a new start command handler
func NewStartHandler(logger *logrus.Logger) *StartHandler {
	return &StartHandler{
		logger: logger,
	}
}

// Handle processes the /start command
func (h *StartHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	welcomeText := `
🎯 *Welcome to TodoboT!*

I'm here to help you manage your family's todo list in this group chat.

*Available Commands:*
• /add <todo> - Add a new todo item
• /list - Show all todos
• /done <id> - Mark todo as completed
• /delete <id> - Delete a todo
• /my - Show your assigned todos
• /help - Show this help message

*Advanced Features:*
• /assign <id> @username - Assign todo to someone
• /priority <id> <high/medium/low> - Set priority
• /deadline <id> <date> - Set deadline
• /comment <id> <text> - Add comment

Get started by adding your first todo with /add!
	`

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
	}).Info("Sent start message")

	return nil
}