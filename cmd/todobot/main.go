package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kerhoff/TodoboT/internal/api"
	"github.com/Kerhoff/TodoboT/internal/config"
	"github.com/Kerhoff/TodoboT/internal/handlers"
	"github.com/Kerhoff/TodoboT/internal/repository/postgres"
	"github.com/Kerhoff/TodoboT/internal/service"
	"github.com/Kerhoff/TodoboT/internal/telegram"
	"github.com/Kerhoff/TodoboT/pkg/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	l := logger.New(cfg.LogLevel)
	l.Info("Starting TodoboT...")

	// Database
	db, err := config.NewDatabase(cfg.DatabaseURL, l)
	if err != nil {
		l.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate("migrations"); err != nil {
		l.Fatalf("Failed to run migrations: %v", err)
	}

	// Repositories
	userRepo := postgres.NewUserRepository(db.DB)
	todoRepo := postgres.NewTodoRepository(db.DB)
	commentRepo := postgres.NewCommentRepository(db.DB)
	familyRepo := postgres.NewFamilyRepository(db.DB)
	calendarRepo := postgres.NewCalendarRepository(db.DB)
	buyingRepo := postgres.NewBuyingListRepository(db.DB)
	wishListRepo := postgres.NewWishListRepository(db.DB)
	reminderRepo := postgres.NewReminderRepository(db.DB)

	// Service layer
	svc := service.New(db.DB, l,
		userRepo, todoRepo, commentRepo, familyRepo,
		calendarRepo, buyingRepo, wishListRepo, reminderRepo,
	)

	// Telegram bot
	bot, err := telegram.NewBot(cfg.TelegramToken, l)
	if err != nil {
		l.Fatalf("Failed to create Telegram bot: %v", err)
	}

	// Register command handlers
	bot.RegisterCommand("start", handlers.NewStartHandler(l))
	bot.RegisterCommand("help", handlers.NewHelpHandler(l))

	// Todo handlers
	bot.RegisterCommand("add", handlers.NewAddHandler(svc, l))
	bot.RegisterCommand("list", handlers.NewListHandler(svc, l))
	bot.RegisterCommand("done", handlers.NewDoneHandler(svc, l))
	bot.RegisterCommand("delete", handlers.NewDeleteHandler(svc, l))
	bot.RegisterCommand("my", handlers.NewMyHandler(svc, l))

	// Calendar handlers
	bot.RegisterCommand("event", handlers.NewCalendarAddHandler(svc, l))
	bot.RegisterCommand("events", handlers.NewCalendarListHandler(svc, l))
	bot.RegisterCommand("delevent", handlers.NewCalendarDeleteHandler(svc, l))

	// Buying list handlers
	bot.RegisterCommand("buy", handlers.NewBuyAddHandler(svc, l))
	bot.RegisterCommand("buylist", handlers.NewBuyListHandler(svc, l))
	bot.RegisterCommand("bought", handlers.NewBuyDoneHandler(svc, l))
	bot.RegisterCommand("buyclear", handlers.NewBuyClearHandler(svc, l))

	// Wish list handlers
	bot.RegisterCommand("wish", handlers.NewWishAddHandler(svc, l))
	bot.RegisterCommand("wishlist", handlers.NewWishListHandler(svc, l))
	bot.RegisterCommand("reserve", handlers.NewWishReserveHandler(svc, l))

	// Reminder handlers
	bot.RegisterCommand("remind", handlers.NewRemindHandler(svc, l))
	bot.RegisterCommand("reminders", handlers.NewRemindersListHandler(svc, l))
	bot.RegisterCommand("delremind", handlers.NewRemindDeleteHandler(svc, l))

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		l.Info("Received shutdown signal...")
		cancel()
	}()

	// Start reminder scheduler
	go svc.StartReminderScheduler(ctx, func(chatID int64, text string) {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.SendRaw(msg)
	})

	// Start HTTP server for web UI
	apiServer := api.NewServer(svc, l)
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: apiServer.Handler(),
	}

	go func() {
		l.Infof("HTTP server listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Errorf("HTTP server error: %v", err)
		}
	}()

	// Start Telegram bot polling
	go func() {
		if err := bot.Start(ctx); err != nil {
			l.Errorf("Bot error: %v", err)
		}
	}()

	l.Info("TodoboT started successfully")

	<-ctx.Done()

	l.Info("Shutting down HTTP server...")
	httpServer.Close()

	l.Info("TodoboT stopped")
}
