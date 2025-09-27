-- Remove signal_id column from analyses table
DROP INDEX IF EXISTS idx_analyses_signal_id;
ALTER TABLE analyses DROP COLUMN signal_id;
