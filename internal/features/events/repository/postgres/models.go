package postgres

import (
	"errors"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type EventsRepositoryImpl struct {
	pool core_pool.Pool
}

func NewEventsRepository(p core_pool.Pool) *EventsRepositoryImpl {
	return &EventsRepositoryImpl{pool: p}
}

func scanEvent(row core_pool.Row) (*domain.Event, error) {
	var e domain.Event
	err := row.Scan(&e.ID, &e.UserID, &e.Title, &e.Description, &e.StartAt, &e.EndAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, mapScanError(err)
	}
	return &e, nil
}

func mapScanError(err error) error {
	if errors.Is(err, core_pool.ErrNoRows) {
		return core_errors.ErrNotFound
	}
	return err
}

func mapEventError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return core_errors.ErrConflict
		}
	}
	return err
}