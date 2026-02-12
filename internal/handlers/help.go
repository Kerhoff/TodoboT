package handlers

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// HelpHandler handles the /help command
type HelpHandler struct {
	logger *logrus.Logger
}

func NewHelpHandler(logger *logrus.Logger) *HelpHandler {
	return &HelpHandler{logger: logger}
}

func (h *HelpHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	helpText := `📚 *TodoboT Help*

*Todos:*
• /add <text> - Add a new todo
• /list - Show pending todos
• /done <id> - Complete a todo
• /delete <id> - Delete a todo
• /my - Show your assigned todos

*Calendar:*
• /event <title> <YYYY-MM-DD> [HH:MM] - Add event
• /events - Show upcoming events
• /delevent <id> - Delete an event

*Shopping List:*
• /buy <item> [x qty] - Add to shopping list
• /buylist - Show shopping list
• /bought <id> - Mark item as bought
• /buyclear - Clear bought items

*Wish Lists:*
• /wish <item> - Add to your wish list
• /wishlist [@user] - View wish lists
• /reserve <id> - Reserve a wish item

*Reminders:*
• /remind <time> <text> - Set reminder
• /reminders - Show your reminders
• /delremind <id> - Delete reminder

_Time formats: 10m, 2h, 1d, 15:30, 2025-01-15 15:30_`

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send help message: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
	}).Info("Sent help message")

	return nil
}
