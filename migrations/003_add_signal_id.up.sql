-- Add signal_id column to analyses table for test signal identification
ALTER TABLE analyses
ADD COLUMN signal_id VARCHAR(50) NOT NULL DEFAULT 'sine_sweep_20_20k';

-- Add index for signal_id if needed for queries
CREATE INDEX IF NOT EXISTS idx_analyses_signal_id ON analyses(signal_id);
