-- 0009_reminder_titles.up.sql
-- Snapshot of the entity title at schedule time, so the push worker
-- can build a notification without joining events/tasks tables.
ALTER TABLE reminders ADD COLUMN title TEXT NOT NULL DEFAULT '';
