package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
)

type buyingListRepository struct {
	db *sql.DB
}

// NewBuyingListRepository creates a new buying list repository
func NewBuyingListRepository(db *sql.DB) repository.BuyingListRepository {
	return &buyingListRepository{db: db}
}

func (r *buyingListRepository) CreateList(ctx context.Context, list *models.BuyingList) (*models.BuyingList, error) {
	query := `
		INSERT INTO buying_lists (family_id, chat_id, name, created_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	list.CreatedAt = now
	list.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query,
		list.FamilyID,
		list.ChatID,
		list.Name,
		list.CreatedByID,
		list.CreatedAt,
		list.UpdatedAt,
	).Scan(&list.ID, &list.CreatedAt, &list.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create buying list: %w", err)
	}

	return list, nil
}

func (r *buyingListRepository) GetListByChatID(ctx context.Context, chatID int64) (*models.BuyingList, error) {
	query := `
		SELECT id, family_id, chat_id, name, created_by_id, created_at, updated_at
		FROM buying_lists
		WHERE chat_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	list := &models.BuyingList{}
	err := r.db.QueryRowContext(ctx, query, chatID).Scan(
		&list.ID,
		&list.FamilyID,
		&list.ChatID,
		&list.Name,
		&list.CreatedByID,
		&list.CreatedAt,
		&list.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get buying list by chat ID: %w", err)
	}

	return list, nil
}

func (r *buyingListRepository) GetListByID(ctx context.Context, id int64) (*models.BuyingList, error) {
	query := `
		SELECT id, family_id, chat_id, name, created_by_id, created_at, updated_at
		FROM buying_lists
		WHERE id = $1`

	list := &models.BuyingList{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&list.ID,
		&list.FamilyID,
		&list.ChatID,
		&list.Name,
		&list.CreatedByID,
		&list.CreatedAt,
		&list.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get buying list by ID: %w", err)
	}

	return list, nil
}

func (r *buyingListRepository) AddItem(ctx context.Context, item *models.BuyingItem) (*models.BuyingItem, error) {
	query := `
		INSERT INTO buying_items (buying_list_id, name, quantity, bought, added_by_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	item.Bought = false
	item.CreatedAt = time.Now()

	err := r.db.QueryRowContext(ctx, query,
		item.BuyingListID,
		item.Name,
		item.Quantity,
		item.Bought,
		item.AddedByID,
		item.CreatedAt,
	).Scan(&item.ID, &item.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to add buying item: %w", err)
	}

	return item, nil
}

func (r *buyingListRepository) GetItems(ctx context.Context, listID int64, onlyUnbought bool) ([]*models.BuyingItem, error) {
	query := `
		SELECT id, buying_list_id, name, quantity, bought, bought_by_id, added_by_id, created_at
		FROM buying_items
		WHERE buying_list_id = $1`

	if onlyUnbought {
		query += " AND bought = false"
	}

	query += " ORDER BY created_at ASC"

	rows, err := r.db.QueryContext(ctx, query, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to query buying items: %w", err)
	}
	defer rows.Close()

	var items []*models.BuyingItem
	for rows.Next() {
		item := &models.BuyingItem{}
		if err := rows.Scan(
			&item.ID,
			&item.BuyingListID,
			&item.Name,
			&item.Quantity,
			&item.Bought,
			&item.BoughtByID,
			&item.AddedByID,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan buying item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *buyingListRepository) MarkBought(ctx context.Context, itemID, boughtByID int64) error {
	query := `
		UPDATE buying_items
		SET bought = true, bought_by_id = $2
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, itemID, boughtByID)
	if err != nil {
		return fmt.Errorf("failed to mark item as bought: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("buying item with ID %d not found", itemID)
	}

	return nil
}

func (r *buyingListRepository) DeleteItem(ctx context.Context, itemID int64) error {
	query := `DELETE FROM buying_items WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete buying item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("buying item with ID %d not found", itemID)
	}

	return nil
}

func (r *buyingListRepository) ClearBought(ctx context.Context, listID int64) error {
	query := `DELETE FROM buying_items WHERE buying_list_id = $1 AND bought = true`

	_, err := r.db.ExecContext(ctx, query, listID)
	if err != nil {
		return fmt.Errorf("failed to clear bought items: %w", err)
	}

	return nil
}
