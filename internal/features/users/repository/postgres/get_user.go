package postgres

import (
	"context"

	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
	"github.com/google/uuid"
)

func (r *UsersRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const sql = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	row := r.pool.QueryRow(ctx, sql, email)
	return scanUser(row)
}

func (r *UsersRepositoryImpl) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	const sql = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	row := r.pool.QueryRow(ctx, sql, username)
	return scanUser(row)
}

func (r *UsersRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const sql = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, sql, id)
	return scanUser(row)
}
