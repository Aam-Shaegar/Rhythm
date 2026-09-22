package postgres

import (
	"context"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/google/uuid"
)

func (r *RemindersRepositoryImpl) CreateReminders(ctx context.Context, reminders []*domain.Reminder) error {
	if len(reminders) == 0 {
		return nil
	}

	const sql = `
		INSERT INTO reminders (id, user_id, entity_type, entity_id, title, remind_at, is_sent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	batch := make([][]any, len(reminders))
	for i, rem := range reminders {
		rem.ID = uuid.New()
		rem.CreatedAt = time.Now().UTC()
		rem.IsSent = false
		batch[i] = []any{rem.ID, rem.UserID, rem.EntityType, rem.EntityID, rem.Title, rem.RemindAt, rem.IsSent, rem.CreatedAt}
	}

	// Execute in a transaction for atomicity
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, args := range batch {
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return mapReminderError(err)
		}
	}

	return tx.Commit(ctx)
}

func (r *RemindersRepositoryImpl) GetPending(ctx context.Context, before time.Time, limit int) ([]*domain.Reminder, error) {
	const sql = `
		SELECT id, user_id, entity_type, entity_id, title, remind_at, is_sent, sent_at, created_at
		FROM reminders
		WHERE is_sent = FALSE AND remind_at <= $1
		ORDER BY remind_at ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, sql, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []*domain.Reminder
	for rows.Next() {
		rem, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reminders, nil
}

func (r *RemindersRepositoryImpl) MarkSent(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	const sql = `
		UPDATE reminders
		SET is_sent = TRUE, sent_at = $2
		WHERE id = $1
	`

	now := time.Now().UTC()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, id := range ids {
		if _, err := tx.Exec(ctx, sql, id, now); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *RemindersRepositoryImpl) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*domain.Reminder, error) {
	const sql = `
		SELECT id, user_id, entity_type, entity_id, title, remind_at, is_sent, sent_at, created_at
		FROM reminders
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY remind_at ASC
	`

	rows, err := r.pool.Query(ctx, sql, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []*domain.Reminder
	for rows.Next() {
		rem, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reminders, nil
}

func (r *RemindersRepositoryImpl) DeleteByEntity(ctx context.Context, entityType string, entityID uuid.UUID) error {
	const sql = `DELETE FROM reminders WHERE entity_type = $1 AND entity_id = $2`

	_, err := r.pool.Exec(ctx, sql, entityType, entityID)
	return err
}

func (r *RemindersRepositoryImpl) UpsertSubscription(ctx context.Context, sub *domain.PushSubscription) error {
	const sql = `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth
	`

	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	sub.CreatedAt = time.Now().UTC()

	_, err := r.pool.Exec(ctx, sql, sub.ID, sub.UserID, sub.Endpoint, sub.P256DH, sub.Auth, sub.CreatedAt)
	return err
}

func (r *RemindersRepositoryImpl) GetSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]*domain.PushSubscription, error) {
	const sql = `
		SELECT id, user_id, endpoint, p256dh, auth, created_at
		FROM push_subscriptions
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, sql, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*domain.PushSubscription
	for rows.Next() {
		s, err := scanPushSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *RemindersRepositoryImpl) DeleteSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error {
	const sql = `DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`

	tag, err := r.pool.Exec(ctx, sql, userID, endpoint)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return core_errors.ErrNotFound
	}
	return nil
}