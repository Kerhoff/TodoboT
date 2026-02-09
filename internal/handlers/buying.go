package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/service"
)

// BuyAddHandler handles the /buy command
type BuyAddHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewBuyAddHandler(svc *service.Service, logger *logrus.Logger) *BuyAddHandler {
	return &BuyAddHandler{svc: svc, logger: logger}
}

func (h *BuyAddHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Usage: /buy <item> [x quantity]")
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()
	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, message.Chat.Title)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}

	// Get or create shopping list for this chat
	list, err := h.svc.Buying.GetListByChatID(ctx, message.Chat.ID)
	if err != nil {
		return fmt.Errorf("get buying list: %w", err)
	}
	if list == nil {
		list = &models.BuyingList{
			FamilyID:    family.ID,
			ChatID:      message.Chat.ID,
			Name:        "Shopping List",
			CreatedByID: user.ID,
		}
		list, err = h.svc.Buying.CreateList(ctx, list)
		if err != nil {
			return fmt.Errorf("create buying list: %w", err)
		}
	}

	// Parse item name and optional quantity
	name := strings.Join(args, " ")
	quantity := ""
	for i, a := range args {
		if a == "x" && i < len(args)-1 {
			name = strings.Join(args[:i], " ")
			quantity = strings.Join(args[i+1:], " ")
			break
		}
	}

	item := &models.BuyingItem{
		BuyingListID: list.ID,
		Name:         name,
		Quantity:     quantity,
		AddedByID:    user.ID,
	}

	item, err = h.svc.Buying.AddItem(ctx, item)
	if err != nil {
		return fmt.Errorf("add buying item: %w", err)
	}

	qtyStr := ""
	if quantity != "" {
		qtyStr = fmt.Sprintf(" (x%s)", quantity)
	}
	text := fmt.Sprintf("🛒 Added to shopping list: *%s*%s", item.Name, qtyStr)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// BuyListHandler handles the /buylist command
type BuyListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewBuyListHandler(svc *service.Service, logger *logrus.Logger) *BuyListHandler {
	return &BuyListHandler{svc: svc, logger: logger}
}

func (h *BuyListHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()
	list, err := h.svc.Buying.GetListByChatID(ctx, message.Chat.ID)
	if err != nil || list == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🛒 No shopping list yet. Add items with /buy")
		bot.Send(msg)
		return nil
	}

	items, err := h.svc.Buying.GetItems(ctx, list.ID, false)
	if err != nil {
		return fmt.Errorf("get buying items: %w", err)
	}

	if len(items) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🛒 Shopping list is empty. Add items with /buy")
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("🛒 *Shopping List:*\n\n")
	for _, item := range items {
		check := "⬜"
		if item.Bought {
			check = "✅"
		}
		qty := ""
		if item.Quantity != "" {
			qty = fmt.Sprintf(" (x%s)", item.Quantity)
		}
		sb.WriteString(fmt.Sprintf("%s #%d: %s%s\n", check, item.ID, item.Name, qty))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// BuyDoneHandler handles the /bought command
type BuyDoneHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewBuyDoneHandler(svc *service.Service, logger *logrus.Logger) *BuyDoneHandler {
	return &BuyDoneHandler{svc: svc, logger: logger}
}

func (h *BuyDoneHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Usage: /bought <id>")
		bot.Send(msg)
		return nil
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Invalid item ID")
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()
	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	if err := h.svc.Buying.MarkBought(ctx, id, user.ID); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Item not found")
		bot.Send(msg)
		return nil
	}

	text := fmt.Sprintf("✅ Item #%d marked as bought", id)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	bot.Send(msg)
	return nil
}

// BuyClearHandler handles the /buyclear command
type BuyClearHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewBuyClearHandler(svc *service.Service, logger *logrus.Logger) *BuyClearHandler {
	return &BuyClearHandler{svc: svc, logger: logger}
}

func (h *BuyClearHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()
	list, err := h.svc.Buying.GetListByChatID(ctx, message.Chat.ID)
	if err != nil || list == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🛒 No shopping list found")
		bot.Send(msg)
		return nil
	}

	if err := h.svc.Buying.ClearBought(ctx, list.ID); err != nil {
		return fmt.Errorf("clear bought items: %w", err)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "🧹 Cleared all bought items from the shopping list")
	bot.Send(msg)
	return nil
}
