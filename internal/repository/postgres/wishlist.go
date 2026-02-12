package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
)

type wishListRepository struct {
	db *sql.DB
}

// NewWishListRepository creates a new wish list repository
func NewWishListRepository(db *sql.DB) repository.WishListRepository {
	return &wishListRepository{db: db}
}

func (r *wishListRepository) CreateList(ctx context.Context, list *models.WishList) (*models.WishList, error) {
	query := `
		INSERT INTO wish_lists (family_id, user_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	list.CreatedAt = now
	list.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query,
		list.FamilyID,
		list.UserID,
		list.Name,
		list.CreatedAt,
		list.UpdatedAt,
	).Scan(&list.ID, &list.CreatedAt, &list.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create wish list: %w", err)
	}

	return list, nil
}

func (r *wishListRepository) GetListByUser(ctx context.Context, userID, familyID int64) (*models.WishList, error) {
	query := `
		SELECT id, family_id, user_id, name, created_at, updated_at
		FROM wish_lists
		WHERE user_id = $1 AND family_id = $2
		ORDER BY created_at DESC
		LIMIT 1`

	list := &models.WishList{}
	err := r.db.QueryRowContext(ctx, query, userID, familyID).Scan(
		&list.ID,
		&list.FamilyID,
		&list.UserID,
		&list.Name,
		&list.CreatedAt,
		&list.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get wish list by user: %w", err)
	}

	return list, nil
}

func (r *wishListRepository) GetListByID(ctx context.Context, id int64) (*models.WishList, error) {
	query := `
		SELECT id, family_id, user_id, name, created_at, updated_at
		FROM wish_lists
		WHERE id = $1`

	list := &models.WishList{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&list.ID,
		&list.FamilyID,
		&list.UserID,
		&list.Name,
		&list.CreatedAt,
		&list.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get wish list by ID: %w", err)
	}

	return list, nil
}

func (r *wishListRepository) GetListsByFamily(ctx context.Context, familyID int64) ([]*models.WishList, error) {
	query := `
		SELECT id, family_id, user_id, name, created_at, updated_at
		FROM wish_lists
		WHERE family_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wish lists by family: %w", err)
	}
	defer rows.Close()

	var lists []*models.WishList
	for rows.Next() {
		list := &models.WishList{}
		if err := rows.Scan(
			&list.ID,
			&list.FamilyID,
			&list.UserID,
			&list.Name,
			&list.CreatedAt,
			&list.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan wish list: %w", err)
		}
		lists = append(lists, list)
	}

	return lists, rows.Err()
}

func (r *wishListRepository) AddItem(ctx context.Context, item *models.WishItem) (*models.WishItem, error) {
	query := `
		INSERT INTO wish_items (wish_list_id, name, url, price, notes, reserved, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	item.Reserved = false
	item.CreatedAt = time.Now()

	err := r.db.QueryRowContext(ctx, query,
		item.WishListID,
		item.Name,
		item.URL,
		item.Price,
		item.Notes,
		item.Reserved,
		item.CreatedAt,
	).Scan(&item.ID, &item.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to add wish item: %w", err)
	}

	return item, nil
}

func (r *wishListRepository) GetItems(ctx context.Context, listID int64) ([]*models.WishItem, error) {
	query := `
		SELECT id, wish_list_id, name, url, price, notes, reserved, reserved_by_id, created_at
		FROM wish_items
		WHERE wish_list_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wish items: %w", err)
	}
	defer rows.Close()

	var items []*models.WishItem
	for rows.Next() {
		item := &models.WishItem{}
		if err := rows.Scan(
			&item.ID,
			&item.WishListID,
			&item.Name,
			&item.URL,
			&item.Price,
			&item.Notes,
			&item.Reserved,
			&item.ReservedByID,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan wish item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *wishListRepository) ReserveItem(ctx context.Context, itemID, reservedByID int64) error {
	query := `
		UPDATE wish_items
		SET reserved = true, reserved_by_id = $2
		WHERE id = $1 AND reserved = false`

	result, err := r.db.ExecContext(ctx, query, itemID, reservedByID)
	if err != nil {
		return fmt.Errorf("failed to reserve wish item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("wish item with ID %d not found or already reserved", itemID)
	}

	return nil
}

func (r *wishListRepository) UnreserveItem(ctx context.Context, itemID int64) error {
	query := `
		UPDATE wish_items
		SET reserved = false, reserved_by_id = NULL
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, itemID)
	if err != nil {
		return fmt.Errorf("failed to unreserve wish item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("wish item with ID %d not found", itemID)
	}

	return nil
}

func (r *wishListRepository) DeleteItem(ctx context.Context, itemID int64) error {
	query := `DELETE FROM wish_items WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete wish item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("wish item with ID %d not found", itemID)
	}

	return nil
}
