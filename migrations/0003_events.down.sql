-- 0003_events.down.sql
DROP TRIGGER IF EXISTS update_events_updated_at ON events;
DROP TABLE IF EXISTS events;