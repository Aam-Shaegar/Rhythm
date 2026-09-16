package postgres

import (
	"errors"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type RemindersRepositoryImpl struct {
	pool core_pool.Pool
}

func NewRemindersRepository(p core_pool.Pool) *RemindersRepositoryImpl {
	return &RemindersRepositoryImpl{pool: p}
}

func scanReminder(row core_pool.Row) (*domain.Reminder, error) {
	var r domain.Reminder
	err := row.Scan(&r.ID, &r.UserID, &r.EntityType, &r.EntityID, &r.RemindAt, &r.IsSent, &r.SentAt, &r.CreatedAt)
	if err != nil {
		return nil, mapScanError(err)
	}
	return &r, nil
}

func mapScanError(err error) error {
	if errors.Is(err, core_pool.ErrNoRows) {
		return core_errors.ErrNotFound
	}
	return err
}

func mapReminderError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return core_errors.ErrConflict
		}
	}
	return err
}