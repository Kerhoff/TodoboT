package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
)

type commentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) repository.CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(ctx context.Context, comment *models.Comment) (*models.Comment, error) {
	query := `INSERT INTO comments (todo_id, user_id, content, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	comment.CreatedAt = time.Now()
	err := r.db.QueryRowContext(ctx, query,
		comment.TodoID, comment.UserID, comment.Content, comment.CreatedAt,
	).Scan(&comment.ID, &comment.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}
	return comment, nil
}

func (r *commentRepository) GetByTodoID(ctx context.Context, todoID int64) ([]*models.Comment, error) {
	query := `SELECT id, todo_id, user_id, content, created_at
		FROM comments WHERE todo_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, todoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		if err := rows.Scan(&c.ID, &c.TodoID, &c.UserID, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *commentRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment %d not found", id)
	}
	return nil
}
