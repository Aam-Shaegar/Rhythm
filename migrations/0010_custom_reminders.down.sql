DROP TABLE reminder_settings;
DROP INDEX idx_reminders_due;
DROP INDEX idx_reminders_entity_time;
DELETE FROM reminders WHERE is_overdue;
ALTER TABLE reminders DROP COLUMN claimed_until;
ALTER TABLE reminders DROP COLUMN is_overdue;
