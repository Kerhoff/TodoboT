package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
)

type todoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) repository.TodoRepository {
	return &todoRepository{db: db}
}

func (r *todoRepository) Create(ctx context.Context, todo *models.Todo) (*models.Todo, error) {
	query := `INSERT INTO todos (title, description, status, priority, deadline, created_by_id, assigned_to_id, chat_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	now := time.Now()
	todo.CreatedAt = now
	todo.UpdatedAt = now
	if todo.Status == "" {
		todo.Status = models.TodoStatusPending
	}
	if todo.Priority == "" {
		todo.Priority = models.TodoPriorityMedium
	}
	err := r.db.QueryRowContext(ctx, query,
		todo.Title, todo.Description, todo.Status, todo.Priority,
		todo.Deadline, todo.CreatedByID, todo.AssignedToID, todo.ChatID,
		todo.CreatedAt, todo.UpdatedAt,
	).Scan(&todo.ID, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create todo: %w", err)
	}
	return todo, nil
}

func (r *todoRepository) GetByID(ctx context.Context, id int64) (*models.Todo, error) {
	query := `SELECT id, title, description, status, priority, deadline, created_by_id, assigned_to_id, chat_id, message_id, created_at, updated_at
		FROM todos WHERE id = $1`
	todo := &models.Todo{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.Priority,
		&todo.Deadline, &todo.CreatedByID, &todo.AssignedToID, &todo.ChatID,
		&todo.MessageID, &todo.CreatedAt, &todo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	return todo, nil
}

func (r *todoRepository) GetByChatID(ctx context.Context, chatID int64, filters repository.TodoFilters) ([]*models.Todo, error) {
	query := `SELECT id, title, description, status, priority, deadline, created_by_id, assigned_to_id, chat_id, message_id, created_at, updated_at
		FROM todos WHERE chat_id = $1`
	args := []interface{}{chatID}
	argIdx := 2

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIdx)
		args = append(args, *filters.Priority)
		argIdx++
	}

	query += " ORDER BY created_at DESC"
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filters.Limit)
		argIdx++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		todo := &models.Todo{}
		if err := rows.Scan(
			&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.Priority,
			&todo.Deadline, &todo.CreatedByID, &todo.AssignedToID, &todo.ChatID,
			&todo.MessageID, &todo.CreatedAt, &todo.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan todo: %w", err)
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

func (r *todoRepository) GetByAssignedUser(ctx context.Context, userID int64, filters repository.TodoFilters) ([]*models.Todo, error) {
	query := `SELECT id, title, description, status, priority, deadline, created_by_id, assigned_to_id, chat_id, message_id, created_at, updated_at
		FROM todos WHERE assigned_to_id = $1`
	args := []interface{}{userID}
	argIdx := 2

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	query += " ORDER BY created_at DESC"
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filters.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query assigned todos: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		todo := &models.Todo{}
		if err := rows.Scan(
			&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.Priority,
			&todo.Deadline, &todo.CreatedByID, &todo.AssignedToID, &todo.ChatID,
			&todo.MessageID, &todo.CreatedAt, &todo.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan todo: %w", err)
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

func (r *todoRepository) Update(ctx context.Context, todo *models.Todo) (*models.Todo, error) {
	query := `UPDATE todos SET title=$2, description=$3, status=$4, priority=$5, deadline=$6, assigned_to_id=$7, updated_at=$8
		WHERE id=$1 RETURNING updated_at`
	todo.UpdatedAt = time.Now()
	err := r.db.QueryRowContext(ctx, query,
		todo.ID, todo.Title, todo.Description, todo.Status, todo.Priority,
		todo.Deadline, todo.AssignedToID, todo.UpdatedAt,
	).Scan(&todo.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update todo: %w", err)
	}
	return todo, nil
}

func (r *todoRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("todo %d not found", id)
	}
	return nil
}
