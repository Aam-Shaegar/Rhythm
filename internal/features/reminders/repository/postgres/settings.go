package postgres

import (
	"context"
	"encoding/json"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/google/uuid"
	"time"
)

func (r *RemindersRepositoryImpl) GetSettings(ctx context.Context, uid uuid.UUID) (domain.ReminderSettings, error) {
	var events, tasks []byte
	err := r.pool.QueryRow(ctx, `SELECT event_offsets, task_offsets FROM reminder_settings WHERE user_id=$1`, uid).Scan(&events, &tasks)
	if err != nil {
		if mapScanError(err) == core_errors.ErrNotFound {
			return domain.DefaultSettings(), nil
		}
		return domain.ReminderSettings{}, err
	}
	var s domain.ReminderSettings
	if err := json.Unmarshal(events, &s.EventOffsets); err != nil {
		return s, err
	}
	if err := json.Unmarshal(tasks, &s.TaskOffsets); err != nil {
		return s, err
	}
	return s, nil
}

func (r *RemindersRepositoryImpl) SaveSettings(ctx context.Context, uid uuid.UUID, input domain.UpdateReminderSettingsInput) (domain.ReminderSettings, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.ReminderSettings{}, err
	}
	defer tx.Rollback(ctx)
	// Lock the account so concurrent PATCH requests merge against the latest settings.
	var locked uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, uid).Scan(&locked); err != nil {
		return domain.ReminderSettings{}, err
	}
	var eventJSON, taskJSON any
	if input.EventOffsets != nil {
		b, _ := json.Marshal(*input.EventOffsets)
		eventJSON = string(b)
	}
	if input.TaskOffsets != nil {
		b, _ := json.Marshal(*input.TaskOffsets)
		taskJSON = string(b)
	}
	var events, tasks []byte
	err = tx.QueryRow(ctx, `INSERT INTO reminder_settings (user_id, event_offsets, task_offsets)
 VALUES ($1, COALESCE($2::jsonb, '[60,0]'), COALESCE($3::jsonb, '[60,0]'))
 ON CONFLICT (user_id) DO UPDATE SET event_offsets=COALESCE($2::jsonb, reminder_settings.event_offsets), task_offsets=COALESCE($3::jsonb, reminder_settings.task_offsets)
 RETURNING event_offsets, task_offsets`, uid, eventJSON, taskJSON).Scan(&events, &tasks)
	if err != nil {
		return domain.ReminderSettings{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM reminders WHERE user_id=$1 AND NOT is_sent AND NOT is_overdue`, uid); err != nil {
		return domain.ReminderSettings{}, err
	}
	if _, err = tx.Exec(ctx, replanSQL, uid); err != nil {
		return domain.ReminderSettings{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ReminderSettings{}, err
	}
	var s domain.ReminderSettings
	if err = json.Unmarshal(events, &s.EventOffsets); err != nil {
		return s, err
	}
	err = json.Unmarshal(tasks, &s.TaskOffsets)
	return s, err
}

const replanSQL = `INSERT INTO reminders (user_id,entity_type,entity_id,title,remind_at)
 SELECT t.user_id,'task',t.id,t.title,t.due_at - (o.value::int * interval '1 minute')
 FROM tasks t JOIN reminder_settings s ON s.user_id=t.user_id
 CROSS JOIN LATERAL jsonb_array_elements_text(s.task_offsets) o
 WHERE t.user_id=$1 AND NOT t.is_completed AND t.due_at - (o.value::int * interval '1 minute') > now()
 UNION ALL
 SELECT e.user_id,'event',e.id,e.title,e.start_at - (o.value::int * interval '1 minute')
 FROM events e JOIN reminder_settings s ON s.user_id=e.user_id
 CROSS JOIN LATERAL jsonb_array_elements_text(s.event_offsets) o
 WHERE e.user_id=$1 AND e.start_at - (o.value::int * interval '1 minute') > now()
 ON CONFLICT (entity_type,entity_id,remind_at) DO NOTHING`

func (r *RemindersRepositoryImpl) QueueOverdue(ctx context.Context, now time.Time) error {
	// Queue only the current two-hour slot after downtime, avoiding a burst of old notifications.
	_, err := r.pool.Exec(ctx, `INSERT INTO reminders (user_id,entity_type,entity_id,title,remind_at,is_overdue)
 SELECT t.user_id,'task',t.id,t.title,
 t.due_at + (floor(extract(epoch FROM ($1::timestamptz - t.due_at))/7200) * interval '2 hours'),TRUE
 FROM tasks t WHERE NOT t.is_completed AND t.due_at + interval '2 hours' <= $1
 AND NOT EXISTS (SELECT 1 FROM reminders r WHERE r.entity_type='task' AND r.entity_id=t.id AND r.is_overdue AND NOT r.is_sent)
 ON CONFLICT (entity_type,entity_id,remind_at) DO NOTHING`, now)
	return err
}
