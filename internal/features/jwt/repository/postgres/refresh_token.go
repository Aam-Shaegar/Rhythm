package postgres

import (
	"context"
	"time"

	core_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	"github.com/google/uuid"
)

type JwtRepositoryImpl struct {
	pool core_pool.Pool
}

func NewJwtRepository(p core_pool.Pool) *JwtRepositoryImpl {
	return &JwtRepositoryImpl{pool: p}
}

func (r *JwtRepositoryImpl) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	const sql = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, sql,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt,
	)
	return err
}

func (r *JwtRepositoryImpl) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const sql = `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	row := r.pool.QueryRow(ctx, sql, tokenHash)
	return scanRefreshToken(row)
}

func (r *JwtRepositoryImpl) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	const sql = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, sql, id, time.Now().UTC())
	return err
}

func (r *JwtRepositoryImpl) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	const sql = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := r.pool.Exec(ctx, sql, userID, time.Now().UTC())
	return err
}