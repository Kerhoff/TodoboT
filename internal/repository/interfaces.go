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

// TodoFilters represents filters for querying todos
type TodoFilters struct {
	Status   *models.TodoStatus
	Priority *models.TodoPriority
	Limit    int
	Offset   int
}