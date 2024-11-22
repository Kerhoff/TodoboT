package main

import (
	"log/slog"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Create a new logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Load the environment variables
	err := godotenv.Load()
	if err != nil {
		logger.Error("Error: {err}")
		os.Exit(1)
	}

	// Get the Telegram bot token from the environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN is not set in the environment")
		os.Exit(1)
	}

	// Create a new bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		logger.Error("Error: {err}")
		os.Exit(1)
	}

	logger.Info("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		logger.Info("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		if _, err := bot.Send(msg); err != nil {
			logger.Error("Error: {err}")
		}
	}
}
