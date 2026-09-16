package postgres

import (
	"context"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
)

func (r *UsersRepositoryImpl) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, sql, email).Scan(&exists)
	return exists, err
}

func (r *UsersRepositoryImpl) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, sql, username).Scan(&exists)
	return exists, err
}

func (r *UsersRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	const sql = `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, updated_at = $5
		WHERE id = $1
	`

	tag, err := r.pool.Exec(ctx, sql,
		user.ID, user.Username, user.Email, user.PasswordHash, user.UpdatedAt,
	)
	if err != nil {
		return mapUserError(err)
	}
	if tag.RowsAffected() == 0 {
		return core_errors.ErrNotFound
	}
	return nil
}