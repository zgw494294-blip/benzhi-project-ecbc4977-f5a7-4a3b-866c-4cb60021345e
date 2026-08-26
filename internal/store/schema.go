package store

const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS tasks (
 id TEXT PRIMARY KEY, wind_farm TEXT NOT NULL, turbine_code TEXT NOT NULL,
 status TEXT NOT NULL, version INTEGER NOT NULL, snapshot BLOB NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS observations (
 id TEXT PRIMARY KEY, task_id TEXT NOT NULL REFERENCES tasks(id), zone_id TEXT NOT NULL,
 sequence INTEGER NOT NULL, captured_at TEXT NOT NULL, payload BLOB NOT NULL,
 UNIQUE(task_id, sequence)
);
CREATE TABLE IF NOT EXISTS audit_events (
 id TEXT PRIMARY KEY, task_id TEXT NOT NULL REFERENCES tasks(id), version INTEGER NOT NULL,
 event_type TEXT NOT NULL, actor TEXT NOT NULL, occurred_at TEXT NOT NULL, payload BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_task ON audit_events(task_id, occurred_at, id);
CREATE TABLE IF NOT EXISTS credentials (
 id TEXT PRIMARY KEY, task_id TEXT NOT NULL UNIQUE REFERENCES tasks(id),
 digest TEXT NOT NULL UNIQUE, payload BLOB NOT NULL, issued_at TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS credential_no_update BEFORE UPDATE ON credentials
BEGIN SELECT RAISE(ABORT, 'release credential is immutable'); END;
CREATE TRIGGER IF NOT EXISTS credential_no_delete BEFORE DELETE ON credentials
BEGIN SELECT RAISE(ABORT, 'release credential is immutable'); END;
CREATE TABLE IF NOT EXISTS idempotency (
 key TEXT PRIMARY KEY, response BLOB NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES(1, CURRENT_TIMESTAMP);
`
