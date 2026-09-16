package postgres

import (
	"context"
	"strconv"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/google/uuid"
)

func (r *EventsRepositoryImpl) Create(ctx context.Context, event *domain.Event) error {
	const sql = `
		INSERT INTO events (id, user_id, title, description, start_at, end_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, sql,
		event.ID, event.UserID, event.Title, event.Description, event.StartAt, event.EndAt, event.CreatedAt, event.UpdatedAt,
	)
	if err != nil {
		return mapEventError(err)
	}
	return nil
}

func (r *EventsRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	const sql = `
		SELECT id, user_id, title, description, start_at, end_at, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, sql, id)
	return scanEvent(row)
}

func (r *EventsRepositoryImpl) GetByUserID(ctx context.Context, filter domain.EventFilter) ([]*domain.Event, error) {
	args := []any{filter.UserID}
	argIdx := 2

	baseSQL := `
		SELECT id, user_id, title, description, start_at, end_at, created_at, updated_at
		FROM events
		WHERE user_id = $1
	`

	if filter.DateFrom != nil {
		baseSQL += " AND start_at >= $" + strconv.Itoa(argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		baseSQL += " AND start_at <= $" + strconv.Itoa(argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	baseSQL += " ORDER BY start_at ASC"

	if filter.Limit > 0 {
		baseSQL += " LIMIT $" + strconv.Itoa(argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	if filter.Offset > 0 {
		baseSQL += " OFFSET $" + strconv.Itoa(argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, baseSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *EventsRepositoryImpl) Update(ctx context.Context, event *domain.Event) error {
	const sql = `
		UPDATE events
		SET title = $2, description = $3, start_at = $4, end_at = $5, updated_at = $6
		WHERE id = $1
	`

	event.UpdatedAt = time.Now().UTC()

	tag, err := r.pool.Exec(ctx, sql,
		event.ID, event.Title, event.Description, event.StartAt, event.EndAt, event.UpdatedAt,
	)
	if err != nil {
		return mapEventError(err)
	}
	if tag.RowsAffected() == 0 {
		return core_errors.ErrNotFound
	}
	return nil
}

func (r *EventsRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	const sql = `DELETE FROM events WHERE id = $1`

	tag, err := r.pool.Exec(ctx, sql, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return core_errors.ErrNotFound
	}
	return nil
}

func (r *EventsRepositoryImpl) GetUpcoming(ctx context.Context, userID uuid.UUID, before time.Time, limit int) ([]*domain.Event, error) {
	const sql = `
		SELECT id, user_id, title, description, start_at, end_at, created_at, updated_at
		FROM events
		WHERE user_id = $1 AND start_at <= $2
		ORDER BY start_at ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, sql, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}