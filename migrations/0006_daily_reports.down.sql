-- 0006_daily_reports.down.sql
DROP TRIGGER IF EXISTS update_daily_reports_updated_at ON daily_reports;
DROP TABLE IF EXISTS daily_reports;