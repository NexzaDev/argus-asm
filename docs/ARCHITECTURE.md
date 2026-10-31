# Architecture

argus-asm is one Go binary with a layered design. The command-line layer calls services. Services use domain types, importers, the probe, and a storage interface. The domain depends on nothing else in the project.

## Packages

| Path | Responsibility | Internal imports |
|---|---|---|
| `cmd/argus-asm` | Process entry point: signal handling, exit code | `cli` |
| `internal/cli` | Argument parsing (standard `flag`), command dispatch, text and JSON output, exit-code mapping, the line console, the lab command | `buildinfo`, `config`, `domain`, `export`, `importer`, `lab`, `probe`, `repository`, `repository/sqlite`, `service` |
| `internal/service` | Use cases: scopes, inventory, services, certificates, findings, lifecycle, tickets, changes, search, notifications, access control, probing, audit | `domain`, `importer`, `probe`, plus the `Store` interface |
| `internal/domain` | Types and rules: hostnames, IP normalization, scope and authorization, severities, lifecycle transitions, risk scoring, RBAC roles and permissions, error sentinels. No I/O. | none |
| `internal/importer` | Bounded parsers for findings CSV, findings JSON, Nmap XML, and PEM certificates | `domain` |
| `internal/probe` | TCP connect checks with bounded ports, timeouts, and concurrency | none |
| `internal/lab` | Loopback-only listener used as a local target. Sends a fixed banner and closes. | `domain` |
| `internal/export` | CSV and JSON writers with formula neutralization | `domain`, `service` |
| `internal/repository/sqlite` | The `Store` implementation: connection setup, migrations, queries, transactions, triggers | `domain`, `repository`, `migrations` |
| `internal/repository` | Shared repository errors (`ErrNotInitialized`) | none |
| `internal/config` | Resolves the database path from flag, environment, or default location | none |
| `internal/buildinfo` | Version and project metadata, overridable with `-ldflags` | none |
| `migrations` | Embedded, versioned SQL: `0001_init.sql`, `0002_lifecycle_and_access.sql` | none |

The dependency direction is enforced by the import graph. `domain` imports no project package, and `probe` and `config` import none either.

## Data flow: findings import

1. `cli` opens the database through `sqlite.Open` without creating it, and builds a `service.Service` with the current actor.
2. `cli` opens the file (or reads standard input) and passes the reader to `service.ImportFindings`.
3. The service checks the `operate` permission. `importer` parses the file within byte, row, and field limits and returns valid rows plus row errors with line numbers.
4. Each row's asset is normalized with `domain.ParseAsset` and must already exist in the inventory. Unknown assets are row errors.
5. If any row has an error, nothing is written. The error summary says so.
6. Otherwise the service computes a dedupe key per row, records one import batch, and applies all observations in one transaction. Critical and high findings get a notification.
7. The service appends an audit entry and returns the counts. `--dry-run` stops before step 6.

## Data flow: active probe

1. `probe run` requires the `operate` permission and an asset that is in scope and has probing allowed by an administrator.
2. Without `--execute`, the command prints the plan and returns without any network call.
3. With `--execute`, `probe` opens TCP connections with the configured timeout and concurrency, and reports open and closed ports.
4. Each open port is stored as a service record with source `tcp-connect` (upserted, so repeated runs do not duplicate it). The service writes an audit entry (`probe.tcp`) with the planned and open counts.

## Storage

- One SQLite file. Migrations create the tables: scopes, assets, services, certificates, findings, import batches, tickets, notifications, users, audit log, relations, and the schema version table.
- Foreign keys are enabled per connection. Assets reference scopes with `ON DELETE RESTRICT`. Findings and other dependents of assets cascade on deletion of the asset.
- Uniqueness constraints protect scope values, asset kind and value, and finding dedupe keys.
- Timestamps are stored as RFC 3339 text in UTC with nanoseconds.
- Migrations run in filename order, each in its own transaction, and are recorded in the version table.
- The store uses a single connection (`SetMaxOpenConns(1)`), serializing writes. A busy timeout of 5 seconds handles contention from other processes.
- The audit log has `BEFORE UPDATE` and `BEFORE DELETE` triggers that raise an error. Each entry stores a SHA-256 hash that covers the entry content and the previous entry's hash.

## Authorization

Every service method takes an `Actor` (name and role) and checks one of three permissions before doing anything: `read`, `operate`, or `administer`. The role table is in `internal/domain/rbac.go`. The first user must be an administrator. Before any user exists, the bootstrap actor `local` has administrator rights so that `init` and the first `user add` can run. The actor is a name supplied by the caller. It is not authenticated. See [THREAT_MODEL.md](THREAT_MODEL.md).

## Concurrency

The database work is single-threaded. The context from `main`, which is cancelled by SIGINT or SIGTERM, is passed to every database call. The probe runs up to the configured number of concurrent connection attempts (default 4). The lab runs one goroutine per listening port, and each accepted connection is closed after the banner is sent. The race detector runs over the whole test suite (`make race`).

## Error model

- Sentinel errors in `domain`: `ErrValidation`, `ErrOutOfScope`, `ErrDuplicate`, `ErrNotFound`, and `ErrForbidden` (defined in `rbac.go`). `errors.Is` matches them through wrapping.
- `repository.ErrNotInitialized` is returned when the database file is missing or has no schema.
- `cli` maps errors to exit codes in one place. See [COMMANDS.md](COMMANDS.md#exit-codes).
- Two kinds of error are intentionally ignored: closing a read-only input file, and closing the database handle on exit. Rolling back after a successful commit is also ignored because it has no effect.

## Decisions

- [ADR 0001](adr/0001-sqlite-local-storage.md): SQLite for local storage.
- [ADR 0002](adr/0002-stdlib-cli-parsing.md): Standard library `flag` for the CLI.
- [ADR 0003](adr/0003-probe-policy-and-lab.md): Active-probe policy and the loopback lab.
- [ADR 0004](adr/0004-line-based-console.md): Line-based console instead of a full-screen TUI.

## Deviations from the proposed layout

The specification proposes `api/`, `tui/`, `pkg/`, `tests/`, `scripts/`, and an HTTP transport. These are absent because the release has no API, no full-screen interface, no public package, and no scripts that need checking in. Empty directories would be scaffolding. Unit tests sit beside the code they test. The CLI is the only transport.

## Not in this architecture

- An HTTP or web interface. These are out of scope for the product.
- A full-screen TUI and a desktop GUI. See [ROADMAP.md](ROADMAP.md).
- Authentication. Adding it means a new trust boundary and a new threat-model review.
