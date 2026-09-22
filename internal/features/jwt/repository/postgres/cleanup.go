package postgres

import (
	"context"
	"errors"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
)

func scanRefreshToken(row core_pool.Row) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.RevokedAt)
	if err != nil {
		if errors.Is(err, core_pool.ErrNoRows) {
			return nil, core_errors.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *JwtRepositoryImpl) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	const sql = `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1
	`

	tag, err := r.pool.Exec(ctx, sql, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
