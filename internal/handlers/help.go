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

// NewHelpHandler creates a new help command handler
func NewHelpHandler(logger *logrus.Logger) *HelpHandler {
	return &HelpHandler{
		logger: logger,
	}
}

// Handle processes the /help command
func (h *HelpHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	helpText := `
📚 *TodoboT Help*

*Basic Commands:*
• \`/start\` - Show welcome message
• \`/help\` - Show this help message
• \`/add <todo>\` - Add a new todo item
• \`/list\` - Show all todos
• \`/done <id>\` - Mark todo as completed
• \`/delete <id>\` - Delete a todo
• \`/my\` - Show your assigned todos

*Advanced Commands:*
• \`/assign <id> @username\` - Assign todo to someone
• \`/priority <id> <high/medium/low>\` - Set priority level
• \`/deadline <id> <date>\` - Set deadline (YYYY-MM-DD format)
• \`/comment <id> <text>\` - Add comment to todo
• \`/completed\` - Show completed todos
• \`/pending\` - Show pending todos

*Examples:*
• \`/add Buy groceries\`
• \`/assign 1 @john\`
• \`/priority 1 high\`
• \`/deadline 1 2024-12-31\`
• \`/done 1\`

*Tips:*
• Use todo ID numbers shown in /list command
• Todos are shared within this group chat
• You can only delete todos you created
• Use @username to mention specific users
	`

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