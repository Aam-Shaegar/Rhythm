CREATE TABLE reminder_settings (
 user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 event_offsets JSONB NOT NULL DEFAULT '[60,0]' CHECK (jsonb_typeof(event_offsets)='array' AND jsonb_array_length(event_offsets)<=5),
 task_offsets JSONB NOT NULL DEFAULT '[60,0]' CHECK (jsonb_typeof(task_offsets)='array' AND jsonb_array_length(task_offsets)<=5)
);
ALTER TABLE reminders ADD COLUMN is_overdue BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE reminders ADD COLUMN claimed_until TIMESTAMPTZ;
-- Consolidate historical duplicates before enforcing one delivery per entity/time.
DELETE FROM reminders a USING reminders b WHERE a.entity_type=b.entity_type AND a.entity_id=b.entity_id AND a.remind_at=b.remind_at AND a.ctid < b.ctid;
CREATE UNIQUE INDEX idx_reminders_entity_time ON reminders(entity_type,entity_id,remind_at);
CREATE INDEX idx_reminders_due ON reminders(remind_at) WHERE NOT is_sent;
INSERT INTO reminder_settings(user_id) SELECT id FROM users;
DELETE FROM reminders WHERE NOT is_sent;
INSERT INTO reminders (user_id,entity_type,entity_id,title,remind_at)
 SELECT t.user_id,'task',t.id,t.title,t.due_at - o.offset_minutes * interval '1 minute'
 FROM tasks t CROSS JOIN (VALUES (60),(0)) o(offset_minutes)
 WHERE NOT t.is_completed AND t.due_at - o.offset_minutes * interval '1 minute' > now()
 UNION ALL
 SELECT e.user_id,'event',e.id,e.title,e.start_at - o.offset_minutes * interval '1 minute'
 FROM events e CROSS JOIN (VALUES (60),(0)) o(offset_minutes)
 WHERE e.start_at - o.offset_minutes * interval '1 minute' > now()
 ON CONFLICT (entity_type,entity_id,remind_at) DO NOTHING;
