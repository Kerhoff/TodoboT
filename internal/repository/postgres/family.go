package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
)

type familyRepository struct {
	db *sql.DB
}

// NewFamilyRepository creates a new family repository
func NewFamilyRepository(db *sql.DB) repository.FamilyRepository {
	return &familyRepository{db: db}
}

func (r *familyRepository) Create(ctx context.Context, family *models.Family) (*models.Family, error) {
	query := `
		INSERT INTO families (chat_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	family.CreatedAt = now
	family.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query,
		family.ChatID,
		family.Name,
		family.CreatedAt,
		family.UpdatedAt,
	).Scan(&family.ID, &family.CreatedAt, &family.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create family: %w", err)
	}

	return family, nil
}

func (r *familyRepository) GetByChatID(ctx context.Context, chatID int64) (*models.Family, error) {
	query := `
		SELECT id, chat_id, name, created_at, updated_at
		FROM families
		WHERE chat_id = $1`

	family := &models.Family{}
	err := r.db.QueryRowContext(ctx, query, chatID).Scan(
		&family.ID,
		&family.ChatID,
		&family.Name,
		&family.CreatedAt,
		&family.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get family by chat ID: %w", err)
	}

	return family, nil
}

func (r *familyRepository) GetByID(ctx context.Context, id int64) (*models.Family, error) {
	query := `
		SELECT id, chat_id, name, created_at, updated_at
		FROM families
		WHERE id = $1`

	family := &models.Family{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&family.ID,
		&family.ChatID,
		&family.Name,
		&family.CreatedAt,
		&family.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get family by ID: %w", err)
	}

	return family, nil
}

func (r *familyRepository) AddMember(ctx context.Context, familyID, userID int64, role string) error {
	query := `
		INSERT INTO family_members (family_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (family_id, user_id) DO UPDATE SET role = $3`

	_, err := r.db.ExecContext(ctx, query, familyID, userID, role, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add family member: %w", err)
	}

	return nil
}

func (r *familyRepository) RemoveMember(ctx context.Context, familyID, userID int64) error {
	query := `DELETE FROM family_members WHERE family_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, familyID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove family member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("member not found in family %d", familyID)
	}

	return nil
}

func (r *familyRepository) GetMembers(ctx context.Context, familyID int64) ([]*models.User, error) {
	query := `
		SELECT u.id, u.telegram_id, u.telegram_username, u.first_name, u.last_name, u.is_active, u.created_at, u.updated_at
		FROM users u
		INNER JOIN family_members fm ON fm.user_id = u.id
		WHERE fm.family_id = $1
		ORDER BY fm.joined_at ASC`

	rows, err := r.db.QueryContext(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query family members: %w", err)
	}
	defer rows.Close()

	var members []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.TelegramUsername,
			&user.FirstName,
			&user.LastName,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan family member: %w", err)
		}
		members = append(members, user)
	}

	return members, rows.Err()
}

func (r *familyRepository) Update(ctx context.Context, family *models.Family) (*models.Family, error) {
	query := `
		UPDATE families
		SET name = $2, updated_at = $3
		WHERE id = $1
		RETURNING updated_at`

	family.UpdatedAt = time.Now()

	err := r.db.QueryRowContext(ctx, query,
		family.ID,
		family.Name,
		family.UpdatedAt,
	).Scan(&family.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update family: %w", err)
	}

	return family, nil
}
