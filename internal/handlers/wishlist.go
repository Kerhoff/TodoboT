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

// WishAddHandler handles the /wish command
type WishAddHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewWishAddHandler(svc *service.Service, logger *logrus.Logger) *WishAddHandler {
	return &WishAddHandler{svc: svc, logger: logger}
}

func (h *WishAddHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Usage: /wish <item name>")
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

	// Get or create wish list for this user in this family
	list, err := h.svc.WishList.GetListByUser(ctx, user.ID, family.ID)
	if err != nil {
		return fmt.Errorf("get wish list: %w", err)
	}
	if list == nil {
		list = &models.WishList{
			FamilyID: family.ID,
			UserID:   user.ID,
			Name:     user.DisplayName() + "'s Wishes",
		}
		list, err = h.svc.WishList.CreateList(ctx, list)
		if err != nil {
			return fmt.Errorf("create wish list: %w", err)
		}
	}

	item := &models.WishItem{
		WishListID: list.ID,
		Name:       strings.Join(args, " "),
	}

	item, err = h.svc.WishList.AddItem(ctx, item)
	if err != nil {
		return fmt.Errorf("add wish item: %w", err)
	}

	text := fmt.Sprintf("🎁 Wish #%d added: *%s*", item.ID, item.Name)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// WishListHandler handles the /wishlist command
type WishListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewWishListHandler(svc *service.Service, logger *logrus.Logger) *WishListHandler {
	return &WishListHandler{svc: svc, logger: logger}
}

func (h *WishListHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, message.Chat.Title)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}

	// If a username is specified, show that user's list
	if len(args) > 0 && strings.HasPrefix(args[0], "@") {
		username := strings.TrimPrefix(args[0], "@")
		targetUser, err := h.svc.Users.GetByUsername(ctx, username)
		if err != nil || targetUser == nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "❌ User not found")
			bot.Send(msg)
			return nil
		}
		return h.showUserWishList(bot, message.Chat.ID, targetUser, family.ID)
	}

	// Show all family wish lists
	lists, err := h.svc.WishList.GetListsByFamily(ctx, family.ID)
	if err != nil {
		return fmt.Errorf("get wish lists: %w", err)
	}

	if len(lists) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🎁 No wish lists yet. Add wishes with /wish")
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("🎁 *Family Wish Lists:*\n\n")
	for _, list := range lists {
		items, err := h.svc.WishList.GetItems(ctx, list.ID)
		if err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("*%s:*\n", list.Name))
		if len(items) == 0 {
			sb.WriteString("  (empty)\n")
		}
		for _, item := range items {
			reserved := ""
			if item.Reserved {
				reserved = " 🔒"
			}
			sb.WriteString(fmt.Sprintf("  #%d: %s%s\n", item.ID, item.Name, reserved))
		}
		sb.WriteString("\n")
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

func (h *WishListHandler) showUserWishList(bot *tgbotapi.BotAPI, chatID int64, user *models.User, familyID int64) error {
	ctx := context.Background()
	list, err := h.svc.WishList.GetListByUser(ctx, user.ID, familyID)
	if err != nil || list == nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🎁 %s has no wish list yet", user.DisplayName()))
		bot.Send(msg)
		return nil
	}

	items, err := h.svc.WishList.GetItems(ctx, list.ID)
	if err != nil {
		return fmt.Errorf("get wish items: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎁 *%s's Wishes:*\n\n", user.FullName()))
	for _, item := range items {
		reserved := ""
		if item.Reserved {
			reserved = " 🔒 Reserved"
		}
		sb.WriteString(fmt.Sprintf("#%d: %s%s\n", item.ID, item.Name, reserved))
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// WishReserveHandler handles the /reserve command
type WishReserveHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

func NewWishReserveHandler(svc *service.Service, logger *logrus.Logger) *WishReserveHandler {
	return &WishReserveHandler{svc: svc, logger: logger}
}

func (h *WishReserveHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Usage: /reserve <wish_id>")
		bot.Send(msg)
		return nil
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Invalid wish ID")
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()
	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	if err := h.svc.WishList.ReserveItem(ctx, id, user.ID); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Could not reserve item")
		bot.Send(msg)
		return nil
	}

	// Send confirmation as private message to avoid spoiling the surprise
	text := fmt.Sprintf("🔒 Wish #%d reserved by you!", id)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	bot.Send(msg)
	return nil
}
