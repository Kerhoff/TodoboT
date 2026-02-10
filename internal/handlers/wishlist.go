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

// ---------------------------------------------------------------------------
// WishAddHandler – /wish <item>
// ---------------------------------------------------------------------------

// WishAddHandler handles the /wish command to add an item to the user's
// personal wish list. If the user does not yet have a wish list for this
// family, one is created automatically.
type WishAddHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewWishAddHandler creates a new WishAddHandler.
func NewWishAddHandler(svc *service.Service, logger *logrus.Logger) *WishAddHandler {
	return &WishAddHandler{svc: svc, logger: logger}
}

// Handle processes the /wish command.
func (h *WishAddHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a wish item.\n"+
				"Usage: `/wish PlayStation 5`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	ctx := context.Background()

	user, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	chatTitle := message.Chat.Title
	if chatTitle == "" {
		chatTitle = message.From.FirstName + "'s family"
	}
	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, chatTitle)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}
	_ = h.svc.EnsureFamilyMember(ctx, family.ID, user.ID)

	// Get or create the user's wish list for this family
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

	itemName := strings.Join(args, " ")
	item := &models.WishItem{
		WishListID: list.ID,
		Name:       itemName,
	}

	item, err = h.svc.WishList.AddItem(ctx, item)
	if err != nil {
		return fmt.Errorf("add wish item: %w", err)
	}

	text := fmt.Sprintf("🎁 *Added to your wish list!*\n\n*#%d* — %s", item.ID, itemName)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"item_id": item.ID,
	}).Info("Wish item added")

	return nil
}

// ---------------------------------------------------------------------------
// WishListHandler – /wishlist [@user]
// ---------------------------------------------------------------------------

// WishListHandler handles the /wishlist command.
//
// Without arguments it shows all family wish lists. When a @username is
// provided it shows that specific user's wish list.
//
// Reservation status (the lock icon) is hidden from the list owner so that
// surprises are not spoiled; other viewers see which items are reserved.
type WishListHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewWishListHandler creates a new WishListHandler.
func NewWishListHandler(svc *service.Service, logger *logrus.Logger) *WishListHandler {
	return &WishListHandler{svc: svc, logger: logger}
}

// Handle processes the /wishlist command.
func (h *WishListHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	ctx := context.Background()

	currentUser, err := h.svc.EnsureUser(ctx, message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	chatTitle := message.Chat.Title
	if chatTitle == "" {
		chatTitle = message.From.FirstName + "'s family"
	}
	family, err := h.svc.EnsureFamily(ctx, message.Chat.ID, chatTitle)
	if err != nil {
		return fmt.Errorf("ensure family: %w", err)
	}

	// If a @username is specified, show that user's wish list
	if len(args) > 0 && strings.HasPrefix(args[0], "@") {
		username := strings.TrimPrefix(args[0], "@")
		targetUser, lookupErr := h.svc.Users.GetByUsername(ctx, username)
		if lookupErr != nil || targetUser == nil {
			msg := tgbotapi.NewMessage(message.Chat.ID,
				fmt.Sprintf("❌ User @%s not found.", username))
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)
			return nil
		}
		return h.showUserWishList(bot, message.Chat.ID, currentUser, targetUser, family.ID)
	}

	// No @user argument — show all family wish lists
	lists, err := h.svc.WishList.GetListsByFamily(ctx, family.ID)
	if err != nil {
		return fmt.Errorf("get wish lists: %w", err)
	}

	if len(lists) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"🎁 *No wish lists yet!*\n\nAdd wishes with `/wish <item>`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("🎁 *Family Wish Lists*\n\n")

	for _, list := range lists {
		items, itemErr := h.svc.WishList.GetItems(ctx, list.ID)
		if itemErr != nil {
			continue
		}

		isOwnList := list.UserID == currentUser.ID
		ownerName := list.Name
		if list.User != nil {
			ownerName = list.User.DisplayName() + "'s Wishes"
		}

		sb.WriteString(fmt.Sprintf("*%s* (%d items)\n", ownerName, len(items)))
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("  *#%d* %s", item.ID, item.Name))
			if item.Price != "" {
				sb.WriteString(fmt.Sprintf(" — _%s_", item.Price))
			}
			// Show reservation status only to non-owners
			if !isOwnList && item.Reserved {
				sb.WriteString(" 🔒")
			}
			sb.WriteString("\n")
		}
		if len(items) == 0 {
			sb.WriteString("  _(empty)_\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("_View a specific list with_ `/wishlist @username`")

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id":    message.Chat.ID,
		"list_count": len(lists),
	}).Info("Listed family wish lists")

	return nil
}

// showUserWishList renders a single user's wish list. Reservation indicators
// are hidden when the viewer is the list owner.
func (h *WishListHandler) showUserWishList(
	bot *tgbotapi.BotAPI,
	chatID int64,
	viewer *models.User,
	owner *models.User,
	familyID int64,
) error {
	ctx := context.Background()
	isOwnList := viewer.ID == owner.ID

	list, err := h.svc.WishList.GetListByUser(ctx, owner.ID, familyID)
	if err != nil || list == nil {
		var emptyText string
		if isOwnList {
			emptyText = "🎁 *You don't have a wish list yet.*\n\nCreate one with `/wish <item>`"
		} else {
			emptyText = fmt.Sprintf("🎁 *%s doesn't have a wish list yet.*", owner.DisplayName())
		}
		msg := tgbotapi.NewMessage(chatID, emptyText)
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	items, err := h.svc.WishList.GetItems(ctx, list.ID)
	if err != nil {
		return fmt.Errorf("get wish items: %w", err)
	}

	if len(items) == 0 {
		var emptyMsg string
		if isOwnList {
			emptyMsg = "🎁 *Your wish list is empty!*\n\nAdd items with `/wish <item>`"
		} else {
			emptyMsg = fmt.Sprintf("🎁 *%s's wish list is empty.*", owner.DisplayName())
		}
		msg := tgbotapi.NewMessage(chatID, emptyMsg)
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	var sb strings.Builder
	if isOwnList {
		sb.WriteString("🎁 *Your Wish List*\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("🎁 *%s's Wish List*\n\n", owner.DisplayName()))
	}

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("%d. *#%d* %s", i+1, item.ID, item.Name))
		if item.URL != "" {
			sb.WriteString(fmt.Sprintf(" ([link](%s))", item.URL))
		}
		if item.Price != "" {
			sb.WriteString(fmt.Sprintf(" — _%s_", item.Price))
		}
		// Show reservation status only to non-owners
		if !isOwnList && item.Reserved {
			sb.WriteString(" 🔒")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\n_%d items_", len(items)))
	if !isOwnList {
		sb.WriteString("\n\n_Use_ `/reserve <id>` _to reserve a gift_")
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id":     chatID,
		"target_user": owner.ID,
		"own_list":    isOwnList,
		"count":       len(items),
	}).Info("Listed user wish list")

	return nil
}

// ---------------------------------------------------------------------------
// WishReserveHandler – /reserve <id>
// ---------------------------------------------------------------------------

// WishReserveHandler handles the /reserve command to reserve a wish item.
// The reservation is hidden from the wish list owner so that it remains a
// surprise.
type WishReserveHandler struct {
	svc    *service.Service
	logger *logrus.Logger
}

// NewWishReserveHandler creates a new WishReserveHandler.
func NewWishReserveHandler(svc *service.Service, logger *logrus.Logger) *WishReserveHandler {
	return &WishReserveHandler{svc: svc, logger: logger}
}

// Handle processes the /reserve command.
func (h *WishReserveHandler) Handle(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) error {
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"❌ Please provide a wish item ID.\n\n"+
				"Usage: `/reserve 5`\n\n"+
				"_View someone's wish list first with_ `/wishlist @username`")
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

	if err = h.svc.WishList.ReserveItem(ctx, itemID, user.ID); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ Could not reserve item *#%d*.\nIt may not exist or is already reserved.", itemID))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	text := fmt.Sprintf("🔒 Item *#%d* reserved!\n\n_The owner won't see who reserved it._", itemID)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	h.logger.WithFields(logrus.Fields{
		"chat_id": message.Chat.ID,
		"user_id": message.From.ID,
		"item_id": itemID,
	}).Info("Wish item reserved")

	return nil
}
