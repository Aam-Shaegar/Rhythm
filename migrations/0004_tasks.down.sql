-- 0004_tasks.down.sql
DROP TRIGGER IF EXISTS update_tasks_updated_at ON tasks;
DROP TABLE IF EXISTS tasks;
DROP TYPE IF EXISTS recurrence_type;