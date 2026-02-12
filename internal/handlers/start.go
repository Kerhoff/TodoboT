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

func NewStartHandler(logger *logrus.Logger) *StartHandler {
	return &StartHandler{logger: logger}
}

func (h *StartHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	welcomeText := `
🎯 *Welcome to TodoboT!*

Your family assistant for tasks, events, shopping, wishes, and reminders.

*Quick Start:*
• /add Buy groceries - Add a todo
• /event Birthday 2025-03-15 - Add event
• /buy Milk x 2 - Add to shopping list
• /wish New headphones - Add to wish list
• /remind 2h Take medicine - Set reminder

Type /help for the full command list!
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
