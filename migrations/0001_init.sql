CREATE TABLE scopes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    kind       TEXT NOT NULL CHECK (kind IN ('domain', 'cidr')),
    value      TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE assets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_id   INTEGER NOT NULL REFERENCES scopes (id) ON DELETE RESTRICT,
    kind       TEXT NOT NULL CHECK (kind IN ('domain', 'ip')),
    value      TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (kind, value)
);

CREATE TABLE findings (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_id    INTEGER NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    severity    TEXT NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    source      TEXT NOT NULL,
    dedupe_key  TEXT NOT NULL UNIQUE,
    first_seen  TEXT NOT NULL,
    last_seen   TEXT NOT NULL
);

CREATE TABLE audit_log (
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    at     TEXT NOT NULL,
    action TEXT NOT NULL,
    detail TEXT NOT NULL
);
