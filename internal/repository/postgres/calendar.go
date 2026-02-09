package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
)

type calendarRepository struct {
	db *sql.DB
}

// NewCalendarRepository creates a new calendar repository
func NewCalendarRepository(db *sql.DB) repository.CalendarRepository {
	return &calendarRepository{db: db}
}

func (r *calendarRepository) Create(ctx context.Context, event *models.CalendarEvent) (*models.CalendarEvent, error) {
	query := `
		INSERT INTO calendar_events (family_id, chat_id, title, description, start_time, end_time, all_day, recurring, location, created_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query,
		event.FamilyID,
		event.ChatID,
		event.Title,
		event.Description,
		event.StartTime,
		event.EndTime,
		event.AllDay,
		event.Recurring,
		event.Location,
		event.CreatedByID,
		event.CreatedAt,
		event.UpdatedAt,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create calendar event: %w", err)
	}

	return event, nil
}

func (r *calendarRepository) GetByID(ctx context.Context, id int64) (*models.CalendarEvent, error) {
	query := `
		SELECT id, family_id, chat_id, title, description, start_time, end_time, all_day, recurring, location, created_by_id, created_at, updated_at
		FROM calendar_events
		WHERE id = $1`

	event := &models.CalendarEvent{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.FamilyID,
		&event.ChatID,
		&event.Title,
		&event.Description,
		&event.StartTime,
		&event.EndTime,
		&event.AllDay,
		&event.Recurring,
		&event.Location,
		&event.CreatedByID,
		&event.CreatedAt,
		&event.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get calendar event: %w", err)
	}

	return event, nil
}

func (r *calendarRepository) GetByChatID(ctx context.Context, chatID int64, filters repository.CalendarFilters) ([]*models.CalendarEvent, error) {
	query := `
		SELECT id, family_id, chat_id, title, description, start_time, end_time, all_day, recurring, location, created_by_id, created_at, updated_at
		FROM calendar_events
		WHERE chat_id = $1`
	args := []interface{}{chatID}
	argIdx := 2

	if filters.From != nil {
		query += fmt.Sprintf(" AND start_time >= $%d", argIdx)
		args = append(args, *filters.From)
		argIdx++
	}
	if filters.To != nil {
		query += fmt.Sprintf(" AND start_time <= $%d", argIdx)
		args = append(args, *filters.To)
		argIdx++
	}

	query += " ORDER BY start_time ASC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filters.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar events: %w", err)
	}
	defer rows.Close()

	var events []*models.CalendarEvent
	for rows.Next() {
		event := &models.CalendarEvent{}
		if err := rows.Scan(
			&event.ID,
			&event.FamilyID,
			&event.ChatID,
			&event.Title,
			&event.Description,
			&event.StartTime,
			&event.EndTime,
			&event.AllDay,
			&event.Recurring,
			&event.Location,
			&event.CreatedByID,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan calendar event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (r *calendarRepository) Update(ctx context.Context, event *models.CalendarEvent) (*models.CalendarEvent, error) {
	query := `
		UPDATE calendar_events
		SET title = $2, description = $3, start_time = $4, end_time = $5, all_day = $6, recurring = $7, location = $8, updated_at = $9
		WHERE id = $1
		RETURNING updated_at`

	event.UpdatedAt = time.Now()

	err := r.db.QueryRowContext(ctx, query,
		event.ID,
		event.Title,
		event.Description,
		event.StartTime,
		event.EndTime,
		event.AllDay,
		event.Recurring,
		event.Location,
		event.UpdatedAt,
	).Scan(&event.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update calendar event: %w", err)
	}

	return event, nil
}

func (r *calendarRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM calendar_events WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete calendar event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("calendar event with ID %d not found", id)
	}

	return nil
}
