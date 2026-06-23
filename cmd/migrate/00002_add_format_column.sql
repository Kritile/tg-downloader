-- Add format column to downloads table
ALTER TABLE downloads ADD COLUMN IF NOT EXISTS format VARCHAR(50) DEFAULT '';

-- Add index on format for faster queries
CREATE INDEX IF NOT EXISTS idx_downloads_format ON downloads(format);
