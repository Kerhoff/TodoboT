package repository

import (
	"context"

	"github.com/Kerhoff/TodoboT/internal/models"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	Delete(ctx context.Context, id int64) error
}

// TodoRepository defines the interface for todo data operations
type TodoRepository interface {
	Create(ctx context.Context, todo *models.Todo) (*models.Todo, error)
	GetByID(ctx context.Context, id int64) (*models.Todo, error)
	GetByChatID(ctx context.Context, chatID int64, filters TodoFilters) ([]*models.Todo, error)
	GetByAssignedUser(ctx context.Context, userID int64, filters TodoFilters) ([]*models.Todo, error)
	Update(ctx context.Context, todo *models.Todo) (*models.Todo, error)
	Delete(ctx context.Context, id int64) error
}

// CommentRepository defines the interface for comment data operations
type CommentRepository interface {
	Create(ctx context.Context, comment *models.Comment) (*models.Comment, error)
	GetByTodoID(ctx context.Context, todoID int64) ([]*models.Comment, error)
	Delete(ctx context.Context, id int64) error
}

// FamilyRepository defines the interface for family data operations
type FamilyRepository interface {
	Create(ctx context.Context, family *models.Family) (*models.Family, error)
	GetByChatID(ctx context.Context, chatID int64) (*models.Family, error)
	GetByID(ctx context.Context, id int64) (*models.Family, error)
	AddMember(ctx context.Context, familyID, userID int64, role string) error
	RemoveMember(ctx context.Context, familyID, userID int64) error
	GetMembers(ctx context.Context, familyID int64) ([]*models.User, error)
	Update(ctx context.Context, family *models.Family) (*models.Family, error)
}

// CalendarRepository defines the interface for calendar event operations
type CalendarRepository interface {
	Create(ctx context.Context, event *models.CalendarEvent) (*models.CalendarEvent, error)
	GetByID(ctx context.Context, id int64) (*models.CalendarEvent, error)
	GetByChatID(ctx context.Context, chatID int64, filters CalendarFilters) ([]*models.CalendarEvent, error)
	Update(ctx context.Context, event *models.CalendarEvent) (*models.CalendarEvent, error)
	Delete(ctx context.Context, id int64) error
}

// BuyingListRepository defines the interface for buying list operations
type BuyingListRepository interface {
	CreateList(ctx context.Context, list *models.BuyingList) (*models.BuyingList, error)
	GetListByChatID(ctx context.Context, chatID int64) (*models.BuyingList, error)
	GetListByID(ctx context.Context, id int64) (*models.BuyingList, error)
	AddItem(ctx context.Context, item *models.BuyingItem) (*models.BuyingItem, error)
	GetItems(ctx context.Context, listID int64, onlyUnbought bool) ([]*models.BuyingItem, error)
	MarkBought(ctx context.Context, itemID, boughtByID int64) error
	DeleteItem(ctx context.Context, itemID int64) error
	ClearBought(ctx context.Context, listID int64) error
}

// WishListRepository defines the interface for wish list operations
type WishListRepository interface {
	CreateList(ctx context.Context, list *models.WishList) (*models.WishList, error)
	GetListByUser(ctx context.Context, userID, familyID int64) (*models.WishList, error)
	GetListByID(ctx context.Context, id int64) (*models.WishList, error)
	GetListsByFamily(ctx context.Context, familyID int64) ([]*models.WishList, error)
	AddItem(ctx context.Context, item *models.WishItem) (*models.WishItem, error)
	GetItems(ctx context.Context, listID int64) ([]*models.WishItem, error)
	ReserveItem(ctx context.Context, itemID, reservedByID int64) error
	UnreserveItem(ctx context.Context, itemID int64) error
	DeleteItem(ctx context.Context, itemID int64) error
}

// ReminderRepository defines the interface for reminder operations
type ReminderRepository interface {
	Create(ctx context.Context, reminder *models.Reminder) (*models.Reminder, error)
	GetByID(ctx context.Context, id int64) (*models.Reminder, error)
	GetByChatID(ctx context.Context, chatID int64) ([]*models.Reminder, error)
	GetByUserID(ctx context.Context, userID int64) ([]*models.Reminder, error)
	GetDue(ctx context.Context) ([]*models.Reminder, error)
	Update(ctx context.Context, reminder *models.Reminder) (*models.Reminder, error)
	Delete(ctx context.Context, id int64) error
	Deactivate(ctx context.Context, id int64) error
}

// TodoFilters represents filters for querying todos
type TodoFilters struct {
	Status   *models.TodoStatus
	Priority *models.TodoPriority
	Limit    int
	Offset   int
}

// CalendarFilters represents filters for querying calendar events
type CalendarFilters struct {
	From  *string
	To    *string
	Limit int
}
