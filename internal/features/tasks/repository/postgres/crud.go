package postgres

import (
	"context"
	"strconv"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

func (r *TasksRepositoryImpl) Create(ctx context.Context, task *domain.Task) error {
	const sql = `
		INSERT INTO tasks (id, user_id, title, description, due_at, is_completed, completed_at, recurrence_type, recurrence_end, parent_task_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.pool.Exec(ctx, sql,
		task.ID, task.UserID, task.Title, task.Description, task.DueAt,
		task.IsCompleted, task.CompletedAt, task.RecurrenceType, task.RecurrenceEnd,
		task.ParentTaskID, task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return mapTaskError(err)
	}
	return nil
}

func (r *TasksRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	const sql = `
		SELECT id, user_id, title, description, due_at, is_completed, completed_at, recurrence_type, recurrence_end, parent_task_id, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, sql, id)
	return scanTask(row)
}

func (r *TasksRepositoryImpl) GetByUserID(ctx context.Context, filter domain.TaskFilter) ([]*domain.Task, error) {
	args := []any{filter.UserID}
	argIdx := 2

	baseSQL := `
		SELECT id, user_id, title, description, due_at, is_completed, completed_at, recurrence_type, recurrence_end, parent_task_id, created_at, updated_at
		FROM tasks
		WHERE user_id = $1
	`

	if filter.DateFrom != nil {
		baseSQL += " AND due_at >= $" + strconv.Itoa(argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		baseSQL += " AND due_at <= $" + strconv.Itoa(argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	if filter.Completed != nil {
		baseSQL += " AND is_completed = $" + strconv.Itoa(argIdx)
		args = append(args, *filter.Completed)
		argIdx++
	}

	if filter.Recurring != nil {
		if *filter.Recurring {
			baseSQL += " AND recurrence_type IS NOT NULL"
		} else {
			baseSQL += " AND recurrence_type IS NULL"
		}
	}

	baseSQL += " ORDER BY due_at ASC"

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

	var tasks []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TasksRepositoryImpl) GetByParentID(ctx context.Context, parentID uuid.UUID) ([]*domain.Task, error) {
	const sql = `
		SELECT id, user_id, title, description, due_at, is_completed, completed_at, recurrence_type, recurrence_end, parent_task_id, created_at, updated_at
		FROM tasks
		WHERE parent_task_id = $1
		ORDER BY due_at ASC
	`

	rows, err := r.pool.Query(ctx, sql, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TasksRepositoryImpl) GetRecurringParents(ctx context.Context, userID uuid.UUID, before time.Time) ([]*domain.Task, error) {
	const sql = `
		SELECT id, user_id, title, description, due_at, is_completed, completed_at, recurrence_type, recurrence_end, parent_task_id, created_at, updated_at
		FROM tasks
		WHERE user_id = $1 AND recurrence_type IS NOT NULL AND parent_task_id IS NULL
		AND (recurrence_end IS NULL OR recurrence_end >= $2)
		ORDER BY due_at ASC
	`

	rows, err := r.pool.Query(ctx, sql, userID, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TasksRepositoryImpl) Update(ctx context.Context, task *domain.Task) error {
	const sql = `
		UPDATE tasks
		SET title = $2, description = $3, due_at = $4, is_completed = $5, completed_at = $6,
		    recurrence_type = $7, recurrence_end = $8, updated_at = $9
		WHERE id = $1
	`

	task.UpdatedAt = time.Now().UTC()

	tag, err := r.pool.Exec(ctx, sql,
		task.ID, task.Title, task.Description, task.DueAt, task.IsCompleted, task.CompletedAt,
		task.RecurrenceType, task.RecurrenceEnd, task.UpdatedAt,
	)
	if err != nil {
		return mapTaskError(err)
	}
	if tag.RowsAffected() == 0 {
		return core_errors.ErrNotFound
	}
	return nil
}

func (r *TasksRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	const sql = `DELETE FROM tasks WHERE id = $1`

	tag, err := r.pool.Exec(ctx, sql, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return core_errors.ErrNotFound
	}
	return nil
}

func (r *TasksRepositoryImpl) GetUsersWithRecurringTasks(ctx context.Context, before time.Time) ([]uuid.UUID, error) {
	const sql = `
		SELECT DISTINCT user_id
		FROM tasks
		WHERE recurrence_type IS NOT NULL AND parent_task_id IS NULL
		AND (recurrence_end IS NULL OR recurrence_end >= $1)
	`

	rows, err := r.pool.Query(ctx, sql, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}