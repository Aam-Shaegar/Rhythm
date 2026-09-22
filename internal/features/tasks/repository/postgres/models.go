package postgres

import (
	"errors"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type TasksRepositoryImpl struct {
	pool core_pool.Pool
}

func NewTasksRepository(p core_pool.Pool) *TasksRepositoryImpl {
	return &TasksRepositoryImpl{pool: p}
}

func scanTask(row core_pool.Row) (*domain.Task, error) {
	var t domain.Task
	err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.DueAt,
		&t.IsCompleted, &t.CompletedAt, &t.RecurrenceType, &t.RecurrenceEnd,
		&t.ParentTaskID, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, mapScanError(err)
	}
	return &t, nil
}

func mapScanError(err error) error {
	if errors.Is(err, core_pool.ErrNoRows) {
		return core_errors.ErrNotFound
	}
	return err
}

func mapTaskError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return core_errors.ErrConflict
		}
	}
	return err
}
