DROP TABLE user_priority_profiles;
DROP TABLE task_priority_snapshots;
DROP INDEX idx_tasks_id_user;
ALTER TABLE tasks DROP COLUMN estimated_minutes;
ALTER TABLE tasks DROP COLUMN importance;
