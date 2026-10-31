# ADR 0001: SQLite for local storage

## Status

Accepted.

## Context

The product must run for €0 on a developer computer, without hosted databases, accounts, or services. It must also support queryable history and durable, auditable records.

## Decision

Use a single SQLite file per user, opened through the `github.com/mattn/go-sqlite3` driver. Schema changes are embedded SQL files in `migrations/`, applied in order and recorded in `schema_migrations`.

## Consequences

- Positive: zero setup, a single file to back up or reset, transactional imports, and standard SQL for queries.
- Negative: the driver uses cgo, so a C compiler is required to build. Concurrent writers are serialized through one connection in the process, which is appropriate for a local tool but not for multi-user service deployment.
- Revisit if a multi-user or networked deployment becomes a requirement. That would be a separate, explicitly approved roadmap item.
