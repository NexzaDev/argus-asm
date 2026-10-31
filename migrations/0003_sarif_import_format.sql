-- foreign_keys=off
CREATE TABLE import_batches_new (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT NOT NULL,
    format     TEXT NOT NULL CHECK (format IN ('csv', 'json', 'sarif')),
    source     TEXT NOT NULL,
    accepted   INTEGER NOT NULL,
    created    INTEGER NOT NULL,
    updated    INTEGER NOT NULL
);

INSERT INTO import_batches_new (id, created_at, format, source, accepted, created, updated)
SELECT id, created_at, format, source, accepted, created, updated FROM import_batches;

DROP TABLE import_batches;

ALTER TABLE import_batches_new RENAME TO import_batches;
