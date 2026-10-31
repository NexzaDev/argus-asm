ALTER TABLE assets ADD COLUMN owner_name TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN verified_by TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN verified_at TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN probe_allowed INTEGER NOT NULL DEFAULT 0 CHECK (probe_allowed IN (0, 1));

ALTER TABLE findings ADD COLUMN status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'accepted', 'resolved'));
ALTER TABLE findings ADD COLUMN status_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE findings ADD COLUMN status_updated_at TEXT NOT NULL DEFAULT '';

ALTER TABLE audit_log ADD COLUMN actor TEXT NOT NULL DEFAULT 'local';
ALTER TABLE audit_log ADD COLUMN prev_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_log ADD COLUMN hash TEXT NOT NULL DEFAULT '';

CREATE TABLE import_batches (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT NOT NULL,
    format     TEXT NOT NULL CHECK (format IN ('csv', 'json')),
    source     TEXT NOT NULL,
    accepted   INTEGER NOT NULL,
    created    INTEGER NOT NULL,
    updated    INTEGER NOT NULL
);

CREATE TABLE finding_observations (
    batch_id   INTEGER NOT NULL REFERENCES import_batches (id) ON DELETE CASCADE,
    finding_id INTEGER NOT NULL REFERENCES findings (id) ON DELETE CASCADE,
    severity   TEXT NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    PRIMARY KEY (batch_id, finding_id)
);

CREATE TABLE services (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_id   INTEGER NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    port       INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    protocol   TEXT NOT NULL CHECK (protocol IN ('tcp', 'udp')),
    name       TEXT NOT NULL DEFAULT '',
    source     TEXT NOT NULL,
    first_seen TEXT NOT NULL,
    last_seen  TEXT NOT NULL,
    UNIQUE (asset_id, port, protocol)
);

CREATE TABLE certificates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_id    INTEGER NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    subject     TEXT NOT NULL,
    issuer      TEXT NOT NULL,
    not_after   TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    UNIQUE (asset_id, fingerprint)
);

CREATE TABLE asset_relations (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    from_asset_id INTEGER NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    to_asset_id   INTEGER NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    kind          TEXT NOT NULL CHECK (kind IN ('resolves_to', 'hosts', 'related_to')),
    created_at    TEXT NOT NULL,
    UNIQUE (from_asset_id, to_asset_id, kind),
    CHECK (from_asset_id <> to_asset_id)
);

CREATE TABLE tickets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    finding_id INTEGER NOT NULL REFERENCES findings (id) ON DELETE CASCADE,
    owner      TEXT NOT NULL,
    status     TEXT NOT NULL CHECK (status IN ('open', 'in_progress', 'done')),
    due_date   TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE notifications (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at      TEXT NOT NULL,
    kind            TEXT NOT NULL,
    message         TEXT NOT NULL,
    dedupe_key      TEXT NOT NULL UNIQUE,
    acknowledged_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    role       TEXT NOT NULL CHECK (role IN ('viewer', 'analyst', 'admin')),
    created_at TEXT NOT NULL
);

CREATE TRIGGER audit_log_no_update BEFORE UPDATE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit log is append-only');
END;

CREATE TRIGGER audit_log_no_delete BEFORE DELETE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit log is append-only');
END;
