package telegram

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// Router handles message routing and command parsing
type Router struct {
	logger   *logrus.Logger
	handlers map[string]CommandHandler
}

// CommandHandler defines the interface for command handlers
type CommandHandler interface {
	Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error
}

// NewRouter creates a new message router
func NewRouter(logger *logrus.Logger) *Router {
	return &Router{
		logger:   logger,
		handlers: make(map[string]CommandHandler),
	}
}

// RegisterCommand registers a command handler
func (r *Router) RegisterCommand(command string, handler CommandHandler) {
	r.handlers[command] = handler
	r.logger.Debugf("Registered command: %s", command)
}

// HandleMessage handles incoming messages
func (r *Router) HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Log the incoming message
	r.logger.WithFields(logrus.Fields{
		"chat_id":    message.Chat.ID,
		"user_id":    message.From.ID,
		"username":   message.From.UserName,
		"message_id": message.MessageID,
		"text":       message.Text,
	}).Info("Received message")

	// Only process text messages
	if message.Text == "" {
		return
	}

	// Check if it's a command
	if !message.IsCommand() {
		return
	}

	command := message.Command()
	args := strings.Fields(message.CommandArguments())

	// Find and execute handler
	if handler, exists := r.handlers[command]; exists {
		if err := handler.Handle(bot, message, args); err != nil {
			r.logger.WithFields(logrus.Fields{
				"command": command,
				"chat_id": message.Chat.ID,
				"user_id": message.From.ID,
				"error":   err,
			}).Error("Command handler failed")

			// Send error message to user
			errorMsg := tgbotapi.NewMessage(message.Chat.ID, "❌ An error occurred while processing your command. Please try again.")
			bot.Send(errorMsg)
		}
	} else {
		// Unknown command
		r.logger.WithFields(logrus.Fields{
			"command": command,
			"chat_id": message.Chat.ID,
			"user_id": message.From.ID,
		}).Warn("Unknown command")

		unknownMsg := tgbotapi.NewMessage(message.Chat.ID, "❓ Unknown command. Use /help to see available commands.")
		bot.Send(unknownMsg)
	}
}

// HandleCallbackQuery handles callback queries from inline keyboards
func (r *Router) HandleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	// Log the callback query
	r.logger.WithFields(logrus.Fields{
		"callback_id": callbackQuery.ID,
		"user_id":     callbackQuery.From.ID,
		"data":        callbackQuery.Data,
	}).Info("Received callback query")

	// Answer the callback query to remove loading state
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	bot.Request(callback)

	// TODO: Implement callback query routing when needed
}