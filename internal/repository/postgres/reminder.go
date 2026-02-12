package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
)

type reminderRepository struct {
	db *sql.DB
}

// NewReminderRepository creates a new reminder repository
func NewReminderRepository(db *sql.DB) repository.ReminderRepository {
	return &reminderRepository{db: db}
}

func (r *reminderRepository) Create(ctx context.Context, reminder *models.Reminder) (*models.Reminder, error) {
	query := `
		INSERT INTO reminders (family_id, chat_id, user_id, text, remind_at, repeat_interval, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	reminder.CreatedAt = now
	reminder.UpdatedAt = now
	reminder.Active = true

	if reminder.Repeat == "" {
		reminder.Repeat = models.ReminderRepeatNone
	}

	err := r.db.QueryRowContext(ctx, query,
		reminder.FamilyID,
		reminder.ChatID,
		reminder.UserID,
		reminder.Text,
		reminder.RemindAt,
		reminder.Repeat,
		reminder.Active,
		reminder.CreatedAt,
		reminder.UpdatedAt,
	).Scan(&reminder.ID, &reminder.CreatedAt, &reminder.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	return reminder, nil
}

func (r *reminderRepository) GetByID(ctx context.Context, id int64) (*models.Reminder, error) {
	query := `
		SELECT id, family_id, chat_id, user_id, text, remind_at, repeat_interval, active, last_sent_at, created_at, updated_at
		FROM reminders
		WHERE id = $1`

	reminder := &models.Reminder{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&reminder.ID,
		&reminder.FamilyID,
		&reminder.ChatID,
		&reminder.UserID,
		&reminder.Text,
		&reminder.RemindAt,
		&reminder.Repeat,
		&reminder.Active,
		&reminder.LastSentAt,
		&reminder.CreatedAt,
		&reminder.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return reminder, nil
}

func (r *reminderRepository) GetByChatID(ctx context.Context, chatID int64) ([]*models.Reminder, error) {
	query := `
		SELECT id, family_id, chat_id, user_id, text, remind_at, repeat_interval, active, last_sent_at, created_at, updated_at
		FROM reminders
		WHERE chat_id = $1
		ORDER BY remind_at ASC`

	rows, err := r.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reminders by chat ID: %w", err)
	}
	defer rows.Close()

	var reminders []*models.Reminder
	for rows.Next() {
		reminder := &models.Reminder{}
		if err := rows.Scan(
			&reminder.ID,
			&reminder.FamilyID,
			&reminder.ChatID,
			&reminder.UserID,
			&reminder.Text,
			&reminder.RemindAt,
			&reminder.Repeat,
			&reminder.Active,
			&reminder.LastSentAt,
			&reminder.CreatedAt,
			&reminder.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	return reminders, rows.Err()
}

func (r *reminderRepository) GetByUserID(ctx context.Context, userID int64) ([]*models.Reminder, error) {
	query := `
		SELECT id, family_id, chat_id, user_id, text, remind_at, repeat_interval, active, last_sent_at, created_at, updated_at
		FROM reminders
		WHERE user_id = $1
		ORDER BY remind_at ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reminders by user ID: %w", err)
	}
	defer rows.Close()

	var reminders []*models.Reminder
	for rows.Next() {
		reminder := &models.Reminder{}
		if err := rows.Scan(
			&reminder.ID,
			&reminder.FamilyID,
			&reminder.ChatID,
			&reminder.UserID,
			&reminder.Text,
			&reminder.RemindAt,
			&reminder.Repeat,
			&reminder.Active,
			&reminder.LastSentAt,
			&reminder.CreatedAt,
			&reminder.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	return reminders, rows.Err()
}

func (r *reminderRepository) GetDue(ctx context.Context) ([]*models.Reminder, error) {
	query := `
		SELECT id, family_id, chat_id, user_id, text, remind_at, repeat_interval, active, last_sent_at, created_at, updated_at
		FROM reminders
		WHERE active = true AND remind_at <= NOW()
		ORDER BY remind_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query due reminders: %w", err)
	}
	defer rows.Close()

	var reminders []*models.Reminder
	for rows.Next() {
		reminder := &models.Reminder{}
		if err := rows.Scan(
			&reminder.ID,
			&reminder.FamilyID,
			&reminder.ChatID,
			&reminder.UserID,
			&reminder.Text,
			&reminder.RemindAt,
			&reminder.Repeat,
			&reminder.Active,
			&reminder.LastSentAt,
			&reminder.CreatedAt,
			&reminder.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan due reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	return reminders, rows.Err()
}

func (r *reminderRepository) Update(ctx context.Context, reminder *models.Reminder) (*models.Reminder, error) {
	query := `
		UPDATE reminders
		SET text = $2, remind_at = $3, repeat_interval = $4, active = $5, last_sent_at = $6, updated_at = $7
		WHERE id = $1
		RETURNING updated_at`

	reminder.UpdatedAt = time.Now()

	err := r.db.QueryRowContext(ctx, query,
		reminder.ID,
		reminder.Text,
		reminder.RemindAt,
		reminder.Repeat,
		reminder.Active,
		reminder.LastSentAt,
		reminder.UpdatedAt,
	).Scan(&reminder.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update reminder: %w", err)
	}

	return reminder, nil
}

func (r *reminderRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM reminders WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reminder with ID %d not found", id)
	}

	return nil
}

func (r *reminderRepository) Deactivate(ctx context.Context, id int64) error {
	query := `
		UPDATE reminders
		SET active = false, updated_at = $2
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to deactivate reminder: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reminder with ID %d not found", id)
	}

	return nil
}
