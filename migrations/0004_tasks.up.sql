-- 0004_tasks.up.sql
CREATE TYPE recurrence_type AS ENUM ('daily', 'weekly', 'monthly', 'yearly');

CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    due_at          TIMESTAMPTZ NOT NULL,
    is_completed    BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at    TIMESTAMPTZ,
    recurrence_type recurrence_type,
    recurrence_end  TIMESTAMPTZ,
    parent_task_id  UUID REFERENCES tasks(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tasks_user_due ON tasks(user_id, due_at);
CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_parent ON tasks(parent_task_id);
CREATE INDEX idx_tasks_recurrence ON tasks(recurrence_type) WHERE recurrence_type IS NOT NULL;
CREATE INDEX idx_tasks_completed ON tasks(user_id, is_completed) WHERE is_completed = FALSE;

CREATE TRIGGER update_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Частичный уникальный индекс: одна открытая задача с recurrence_type на пользователя (опционально, для защиты от дублей)
-- CREATE UNIQUE INDEX idx_tasks_one_active_recurring ON tasks(user_id, recurrence_type) WHERE recurrence_type IS NOT NULL AND is_completed = FALSE;