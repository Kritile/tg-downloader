ALTER TABLE users ADD COLUMN IF NOT EXISTS auto_best_download BOOLEAN DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_users_auto_best_download ON users(auto_best_download);
