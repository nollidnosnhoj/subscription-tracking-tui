CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Default month cutoff day is 1 (start of month)
INSERT OR IGNORE INTO config (key, value) VALUES ('month_cutoff_day', '1');
