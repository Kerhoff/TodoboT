package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// Bot wraps the Telegram bot API
type Bot struct {
	api    *tgbotapi.BotAPI
	logger *logrus.Logger
	router *Router
}

// NewBot creates a new Telegram bot instance
func NewBot(token string, logger *logrus.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	logger.Infof("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api:    api,
		logger: logger,
		router: NewRouter(logger),
	}, nil
}

// SetWebhook sets up webhook for the bot
func (b *Bot) SetWebhook(webhookURL string) error {
	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	_, err = b.api.Request(wh)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	b.logger.Infof("Webhook set to %s", webhookURL)
	return nil
}

// Start starts the bot with long polling
func (b *Bot) Start(ctx context.Context) error {
	// Delete webhook if exists and use polling
	_, err := b.api.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	b.logger.Info("Bot started with long polling")

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Stopping bot...")
			b.api.StopReceivingUpdates()
			return nil
		case update := <-updates:
			go b.handleUpdate(update)
		}
	}
}

// HandleWebhook handles incoming webhook updates
func (b *Bot) HandleWebhook(update tgbotapi.Update) {
	go b.handleUpdate(update)
}

// handleUpdate processes incoming updates
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Errorf("Panic in update handler: %v", r)
		}
	}()

	if update.Message != nil {
		b.router.HandleMessage(b.api, update.Message)
	} else if update.CallbackQuery != nil {
		b.router.HandleCallbackQuery(b.api, update.CallbackQuery)
	}
}

// SendMessage sends a message to a chat
func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := b.api.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// EditMessage edits an existing message
func (b *Bot) EditMessage(chatID int64, messageID int, text string) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := b.api.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

// DeleteMessage deletes a message
func (b *Bot) DeleteMessage(chatID int64, messageID int) error {
	msg := tgbotapi.NewDeleteMessage(chatID, messageID)

	_, err := b.api.Request(msg)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// RegisterCommand registers a command handler on the router
func (b *Bot) RegisterCommand(command string, handler CommandHandler) {
	b.router.RegisterCommand(command, handler)
}

// SendRaw sends a raw tgbotapi.Chattable message
func (b *Bot) SendRaw(c tgbotapi.Chattable) {
	if _, err := b.api.Send(c); err != nil {
		b.logger.Errorf("Failed to send message: %v", err)
	}
}