package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/service"
)

var quantityRegex = regexp.MustCompile(`^x(\d+)$`)

// ---------------------------------------------------------------------------
// BuyAddHandler – /buy <item> [x quantity]
// ---------------------------------------------------------------------------

// BuyAddHandler handles the /buy command to add an item to the shopping list.
// If no shopping list exists for the chat, one is created automatically.
// An optional quantity suffix like "x2" can be appended at the end.
type BuyAddHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewBuyAddHandler creates a new BuyAddHandler.
func NewBuyAddHandler(svc *service.Service, logger *logrus.Logger) *BuyAddHandler {
	return &BuyAddHandler{svc: svc, logger: logger}
}

// Handle processes the /buy command.
func (h *BuyAddHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide an item name.\n\n"+
				"*Usage:*\n"+
				"`/buy Milk x2`\n"+
				"`/buy Whole wheat bread`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Parse optional quantity suffix (e.g. "x2", "x12")
	var itemName, quantity string
	lastArg := args[len(args)-1]

	if matches := quantityRegex.FindStringSubmatch(lastArg); matches != nil && len(args) > 1 {
		quantity = matches[1]
		itemName = strings.Join(args[:len(args)-1], " ")
	} else {
		itemName = strings.Join(args, " ")
		quantity = "1"
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

	// Get or auto-create the shopping list for this chat
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

	item := &models.BuyingItem{
		BuyingListID: list.ID,
		Name:         itemName,
		Quantity:     quantity,
		AddedByID:    user.ID,
	}

	item, err = h.svc.Buying.AddItem(ctx, item)
	if err != nil {
		return fmt.Errorf("add buying item: %w", err)
	}

	var quantityDisplay string
	if quantity != "1" {
		quantityDisplay = fmt.Sprintf(" (x%s)", quantity)
	}

	text := fmt.Sprintf("🛒 *Added to shopping list!*\n\n⬜ *#%d* — %s%s", item.ID, itemName, quantityDisplay)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"item_id": item.ID,
	}).Info("Item added to shopping list")

	return nil
}

// ---------------------------------------------------------------------------
// BuyListHandler – /buylist
// ---------------------------------------------------------------------------

// BuyListHandler handles the /buylist command to display the shopping list,
// showing both bought and unbought items with their status.
type BuyListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewBuyListHandler creates a new BuyListHandler.
func NewBuyListHandler(svc *service.Service, logger *logrus.Logger) *BuyListHandler {
	return &BuyListHandler{svc: svc, logger: logger}
}

// Handle processes the /buylist command.
func (h *BuyListHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	list, err := h.svc.Buying.GetListByChatID(ctx, message.Chat.ID)
	if err != nil || list == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"🛒 *No shopping list yet!*\n\nStart one with `/buy <item>`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Get all items (both bought and unbought)
	items, err := h.svc.Buying.GetItems(ctx, list.ID, false)
	if err != nil {
		return fmt.Errorf("get buying items: %w", err)
	}

	if len(items) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"🛒 *Shopping list is empty!*\n\nAdd items with `/buy <item>`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("🛒 *Shopping List*\n\n")

	var unboughtCount, boughtCount int
	for _, item := range items {
		var quantityDisplay string
		if item.Quantity != "" && item.Quantity != "1" {
			quantityDisplay = fmt.Sprintf(" (x%s)", item.Quantity)
		}

		if item.Bought {
			boughtCount++
			boughtBy := ""
			if item.BoughtBy != nil {
				boughtBy = fmt.Sprintf(" — _by %s_", item.BoughtBy.DisplayName())
			}
			sb.WriteString(fmt.Sprintf("✅ ~%s%s~%s\n", item.Name, quantityDisplay, boughtBy))
		} else {
			unboughtCount++
			sb.WriteString(fmt.Sprintf("⬜ *#%d* %s%s\n", item.ID, item.Name, quantityDisplay))
		}
	}

	sb.WriteString(fmt.Sprintf("\n_%d remaining, %d bought_", unboughtCount, boughtCount))
	if boughtCount > 0 {
		sb.WriteString("\n\n_Use_ `/buyclear` _to remove bought items_")
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"total":   len(items),
	}).Info("Listed shopping list")

	return nil
}

// ---------------------------------------------------------------------------
// BuyDoneHandler – /bought <id>
// ---------------------------------------------------------------------------

// BuyDoneHandler handles the /bought command to mark an item as bought.
type BuyDoneHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewBuyDoneHandler creates a new BuyDoneHandler.
func NewBuyDoneHandler(svc *service.Service, logger *logrus.Logger) *BuyDoneHandler {
	return &BuyDoneHandler{svc: svc, logger: logger}
}

// Handle processes the /bought command.
func (h *BuyDoneHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide an item ID.\nUsage: `/bought 3`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	itemID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Invalid ID. Please provide a numeric item ID.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	if err = h.svc.Buying.MarkBought(ctx, itemID, user.ID); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Could not mark item *#%d* as bought. It may not exist.", itemID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	text := fmt.Sprintf("✅ Item *#%d* marked as bought!", itemID)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"item_id": itemID,
	}).Info("Item marked as bought")

	return nil
}

// ---------------------------------------------------------------------------
// BuyClearHandler – /buyclear
// ---------------------------------------------------------------------------

// BuyClearHandler handles the /buyclear command to clear all bought items
// from the shopping list.
type BuyClearHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewBuyClearHandler creates a new BuyClearHandler.
func NewBuyClearHandler(svc *service.Service, logger *logrus.Logger) *BuyClearHandler {
	return &BuyClearHandler{svc: svc, logger: logger}
}

// Handle processes the /buyclear command.
func (h *BuyClearHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	list, err := h.svc.Buying.GetListByChatID(ctx, message.Chat.ID)
	if err != nil || list == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ No shopping list found for this chat.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	if err = h.svc.Buying.ClearBought(ctx, list.ID); err != nil {
		return fmt.Errorf("clear bought items: %w", err)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID,
		"🧹 All bought items have been cleared from the shopping list!")
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"list_id": list.ID,
	}).Info("Cleared bought items")

	return nil
}
