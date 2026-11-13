ALTER TABLE tasks ADD COLUMN importance SMALLINT NOT NULL DEFAULT 2 CHECK(importance BETWEEN 1 AND 3);
ALTER TABLE tasks ADD COLUMN estimated_minutes INT NOT NULL DEFAULT 30 CHECK(estimated_minutes BETWEEN 5 AND 480);
CREATE UNIQUE INDEX idx_tasks_id_user ON tasks(id,user_id);
CREATE TABLE task_priority_snapshots (
 task_id UUID PRIMARY KEY,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 revision CHAR(64) NOT NULL,
 model_version TEXT NOT NULL,
 features JSONB NOT NULL CHECK(jsonb_typeof(features)='array' AND jsonb_array_length(features)=18),
 observed_at TIMESTAMPTZ NOT NULL,
 due_at TIMESTAMPTZ NOT NULL,
 CHECK(observed_at < due_at),
 FOREIGN KEY(task_id,user_id) REFERENCES tasks(id,user_id) ON DELETE CASCADE
);
CREATE INDEX idx_priority_snapshots_user ON task_priority_snapshots(user_id);
CREATE TABLE user_priority_profiles (
 user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 model_version TEXT NOT NULL,
 dataset_hash CHAR(64) NOT NULL,
 delta JSONB NOT NULL CHECK(jsonb_typeof(delta)='array' AND jsonb_array_length(delta)=18),
 samples INT NOT NULL CHECK(samples BETWEEN 0 AND 500),
 trained_at TIMESTAMPTZ NOT NULL
);
