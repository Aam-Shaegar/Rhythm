package postgres

import (
	"errors"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type UsersRepositoryImpl struct {
	pool core_pool.Pool
}

func NewUsersRepository(p core_pool.Pool) *UsersRepositoryImpl {
	return &UsersRepositoryImpl{pool: p}
}

func scanUser(row core_pool.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, mapScanError(err)
	}
	return &u, nil
}

func mapScanError(err error) error {
	if errors.Is(err, core_pool.ErrNoRows) {
		return core_errors.ErrNotFound
	}
	return err
}

func mapUserError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return core_errors.ErrConflict
		}
	}
	return err
}
