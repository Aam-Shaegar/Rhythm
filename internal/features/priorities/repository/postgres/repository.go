package postgres

import (
	"context"
	"encoding/json"
	"errors"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/domain"
	tasks "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

type Repository struct{ pool pool.Pool }

func New(p pool.Pool) *Repository { return &Repository{pool: p} }
func (r *Repository) WithUser(ctx context.Context, uid uuid.UUID, fn func(domain.Session) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var locked uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, uid).Scan(&locked); err != nil {
		if errors.Is(err, pool.ErrNoRows) {
			return core_errors.ErrUnauthorized
		}
		return err
	}
	if err = fn(&session{tx: tx, uid: uid}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) UserIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT user_id FROM tasks UNION SELECT user_id FROM user_priority_profiles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		ids = append(ids, uid)
	}
	return ids, rows.Err()
}

type session struct {
	tx  pool.Tx
	uid uuid.UUID
}

func (s *session) Tasks(ctx context.Context) ([]*tasks.Task, error) {
	rows, err := s.tx.Query(ctx, `SELECT id,user_id,title,description,due_at,is_completed,completed_at,recurrence_type,recurrence_end,parent_task_id,created_at,updated_at,importance,estimated_minutes FROM tasks WHERE user_id=$1 ORDER BY created_at,id FOR SHARE`, s.uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*tasks.Task{}
	for rows.Next() {
		t := &tasks.Task{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.DueAt, &t.IsCompleted, &t.CompletedAt, &t.RecurrenceType, &t.RecurrenceEnd, &t.ParentTaskID, &t.CreatedAt, &t.UpdatedAt, &t.Importance, &t.EstimatedMinutes); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (s *session) Observations(ctx context.Context) (map[uuid.UUID]domain.Observation, error) {
	rows, err := s.tx.Query(ctx, `SELECT task_id,revision,model_version,features,observed_at,due_at FROM task_priority_snapshots WHERE user_id=$1`, s.uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]domain.Observation{}
	for rows.Next() {
		var o domain.Observation
		var data []byte
		if err := rows.Scan(&o.TaskID, &o.Revision, &o.ModelVersion, &data, &o.ObservedAt, &o.DueAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &o.Features); err != nil {
			return nil, err
		}
		out[o.TaskID] = o
	}
	return out, rows.Err()
}
func (s *session) Profile(ctx context.Context) (domain.Profile, error) {
	var p domain.Profile
	var data []byte
	err := s.tx.QueryRow(ctx, `SELECT model_version,dataset_hash,delta,samples,trained_at FROM user_priority_profiles WHERE user_id=$1`, s.uid).Scan(&p.ModelVersion, &p.DatasetHash, &data, &p.Samples, &p.TrainedAt)
	if errors.Is(err, pool.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(data, &p.Delta)
	return p, err
}
func (s *session) SaveObservation(ctx context.Context, o domain.Observation) error {
	data, err := json.Marshal(o.Features)
	if err != nil {
		return err
	}
	_, err = s.tx.Exec(ctx, `INSERT INTO task_priority_snapshots(task_id,user_id,revision,model_version,features,observed_at,due_at) VALUES($1,$2,$3,$4,$5::jsonb,$6,$7)
 ON CONFLICT(task_id) DO UPDATE SET revision=EXCLUDED.revision,model_version=EXCLUDED.model_version,features=EXCLUDED.features,observed_at=EXCLUDED.observed_at,due_at=EXCLUDED.due_at WHERE task_priority_snapshots.user_id=EXCLUDED.user_id`, o.TaskID, s.uid, o.Revision, o.ModelVersion, string(data), o.ObservedAt, o.DueAt)
	return err
}
func (s *session) DeleteObservation(ctx context.Context, id uuid.UUID) error {
	_, err := s.tx.Exec(ctx, `DELETE FROM task_priority_snapshots WHERE user_id=$1 AND task_id=$2`, s.uid, id)
	return err
}
func (s *session) SaveProfile(ctx context.Context, p domain.Profile) error {
	data, err := json.Marshal(p.Delta)
	if err != nil {
		return err
	}
	_, err = s.tx.Exec(ctx, `INSERT INTO user_priority_profiles(user_id,model_version,dataset_hash,delta,samples,trained_at) VALUES($1,$2,$3,$4::jsonb,$5,$6)
 ON CONFLICT(user_id) DO UPDATE SET model_version=EXCLUDED.model_version,dataset_hash=EXCLUDED.dataset_hash,delta=EXCLUDED.delta,samples=EXCLUDED.samples,trained_at=EXCLUDED.trained_at`, s.uid, p.ModelVersion, p.DatasetHash, string(data), p.Samples, p.TrainedAt)
	return err
}
