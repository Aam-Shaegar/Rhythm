-- 0007_fix_refresh_tokens_trigger.up.sql
-- refresh_tokens has no updated_at column, but 0002 created a trigger
-- calling update_updated_at_column() (NEW.updated_at = now()).
-- Every UPDATE on refresh_tokens (token rotation on refresh) fails with:
--   record "new" has no field "updated_at".
-- Drop the broken trigger.
DROP TRIGGER IF EXISTS update_refresh_tokens_updated_at ON refresh_tokens;
