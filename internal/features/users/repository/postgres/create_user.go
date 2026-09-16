package postgres

import (
	"context"

	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
)

func (r *UsersRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	const sql = `
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, sql,
		user.ID, user.Username, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return mapUserError(err)
	}
	return nil
}