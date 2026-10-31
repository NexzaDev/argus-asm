# Changelog

All notable changes are recorded here. The format follows Keep a Changelog. Semantic versioning applies once a release is tagged.

## [Unreleased]

- Docs: SECURITY.md now uses GitHub private vulnerability reporting with a backup email, the placeholder and private-repository wording are removed, and the README describes the desktop GUI, the launcher, the opt-in notifier, and the pre-release downloads.

- Fixed: the "All notifications" filter label was cut off. The filter and severity selects on Notifications now reserve the width of their longest label (`TestNotificationFilterLabelsFitTheirSelects`).
- Release archives now include `licenses/` with the license files of the third-party modules in each platform's GUI build (`scripts/collect-licenses.sh`).
- `THIRD_PARTY_NOTICES.md` lists the transitive Fyne modules with the license read from each module's LICENSE file, and which ones are in the default build on each platform.
- Release workflow (manual, `release.yml`): builds the GUI for Linux, macOS, and Windows with `fyne.io/tools/cmd/fyne@v1.7.4`, writes `SHA256SUMS`, and creates a draft GitHub Release. Not run on GitHub yet. Binaries are unsigned. See [docs/RELEASING.md](docs/RELEASING.md) and ADR 0005.
- Icon asset checks (`TestIconAssetsAreWellFormed`) for ICO sizes, ICNS chunks, and PNG dimensions.
- `go-sum.yml` runs only when started by hand. It is still enabled on GitHub: disabling it returned HTTP 403 to the sandbox token, so a maintainer must disable it.
- Background notifier (opt-in): `argus-asm notifier run|install|uninstall|status`. Per-user autostart only (Windows Startup folder, macOS LaunchAgent, Linux XDG autostart), read-only database access, no network calls, minimal text by default. Install requires `--yes`. See [docs/COMMANDS.md](docs/COMMANDS.md).
- GUI: unread notification count button in the status bar, severity column and filter on Notifications, Mark as read.
- Security: the GUI now uses `theme.Color(theme.ColorNameForeground)` instead of the deprecated `theme.ForegroundColor`.

### Added (round 5, milestones 1 and 2)
- Desktop GUI built with Fyne (default build; `-tags nogui` excludes it). Launcher chooser, `--ui gui|console|cli`, `ARGUS_UI`, `config set-ui`, `config reset-ui`, headless fallback to the console.
- Dashboard: sidebar with twelve categories, keyboard shortcuts, collapsible sidebar, tables with filter, sort, detail panel, copy, CSV export, and density, Overview with risk tiers, About with usage search, shortcuts, diagnostics preview, and licenses.
- Manual `go.sum` refresh workflow (`workflow_dispatch` only).
- CI: GUI build on Ubuntu, macOS, Windows; design screenshots artifact.

### Added

- Launcher with `--ui console|cli`, `ARGUS_UI`, and `config show|set-ui|reset-ui`. The stored preference lives in the per-user config directory.
- Command-line application `argus-asm` with setup, scope, inventory, services, certificates, findings, lifecycle, tickets, change detection, notifications, search, export, users, audit, probe, lab, and console commands. See [docs/COMMANDS.md](docs/COMMANDS.md).
- Scope enforcement: domain scopes cover the name and its subdomains, CIDR scopes cover their range (IPv4 `/16` or narrower, IPv6 `/32` or narrower). Assets outside every scope are refused.
- Findings importers for CSV, JSON, and Nmap XML, and a PEM certificate importer. Imports are validated completely before any write, support `--dry-run`, and are deduplicated.
- Risk scoring with factors (`findings list --explain`, `findings show`). Accepted findings score 0, and accepting a finding requires a reason.
- Remediation tickets with owners, due dates, and an overdue filter. Change detection between two imports from one source (`findings changes`).
- Notifications for critical and high findings, certificates expiring within 30 days, and overdue tickets. They are stored locally only.
- Allowlisted active checks: TCP connect only, off by default, limited to in-scope assets an administrator has allowed, and performed only with `--execute`. Open ports are stored as services with source `tcp-connect`.
- `lab serve`: a loopback-only TCP listener on unprivileged ports for trying probes safely.
- Roles (viewer, analyst, administrator) enforced in the service layer. The actor is a claimed name, not an authenticated identity.
- Append-only audit log enforced by database triggers, with a SHA-256 hash chain and `audit verify`.
- Export of findings and assets to CSV and JSON, with formula neutralization.
- Synthetic demo fixtures in `testdata/demo/` (RFC 2606 names and RFC 5737 addresses) and a `make demo` target.
- Fuzz targets for the CSV, JSON, and Nmap XML parsers, and a fuzz smoke target in `make fuzz-smoke`.
- Continuous integration for formatting, vet, race-enabled tests, a build, and a smoke test on Linux, macOS, and Windows, plus a static-analysis and vulnerability-scan job.
- SARIF 2.1.0 import (`findings import --format sarif`), with a synthetic fixture in `testdata/demo/findings.sarif` and a fuzz target.
- `notify deliver --to FILE`: appends undelivered notifications to a local file as JSON lines, mode `0600`, with `--dry-run`. Delivery is at-least-once and never uses the network.
- `db seal` and `db unseal`: encrypted copies of the database. AES-256-GCM with a key derived from a passphrase by PBKDF2-HMAC-SHA256 (600,000 iterations). The passphrase is read from standard input only. See [docs/SECURITY.md](docs/SECURITY.md#encrypted-copies).
- `make dist` and a `release-artifacts` CI job that build a per-platform archive with LICENSE, README, and THIRD_PARTY_NOTICES, plus a SHA-256 file. Nothing is published.

### Changed

- Schema version 2 (`0002_lifecycle_and_access.sql`) adds tables for import batches, services, certificates, asset relations, tickets, notifications, users, and the append-only audit triggers. Existing schema 1 databases are upgraded by `init`. Audit entries written before the upgrade have no hash and are reported as legacy.
- CI test jobs run on the `go.mod` toolchain, Go 1.27.3. The `static-analysis` job uses staticcheck 2026.2.2, and the `vulnerability-scan` job uses govulncheck v1.9.0 on Go 1.27.3. The separate `minimum-go` job was removed, because the test matrix already runs the exact version. Each Go job records its toolchain version.

- Schema version 3 (`0003_sarif_import_format.sql`) allows the `sarif` import format. Schema version 4 (`0004_notification_delivery.sql`) records when a notification was delivered. Both migrations run on `init`.

### Fixed

- Windows CI failed the gofmt gate because Git's `core.autocrlf` wrote CRLF line endings. `.gitattributes` now forces LF for source files.
- The `go` directive is now `1.27.3`, up from `1.24.0`. Standard-library vulnerabilities reachable from this code (GO-2026-6088, GO-2026-5972, GO-2026-4971, GO-2026-4601, and GO-2026-4341 on Go 1.24.11 and earlier) are not present in Go 1.27.3, and govulncheck reports no findings on it. See [docs/SECURITY.md](docs/SECURITY.md#vulnerability-scanning).
- Added tests that Nmap XML and PEM inputs at exactly the byte limit are accepted, and one byte over it is refused.
- The CI smoke test covers SARIF import, notification delivery (including a second run that delivers zero), and a seal and unseal round trip.

### Security

- Go 1.25 is out of upstream support since Go 1.27.0 was released, and the build now uses Go 1.27.3. The Go 1.25 line had 14 known standard-library findings that this code does not call; they are not present in Go 1.27.3. See [docs/SECURITY.md](docs/SECURITY.md#vulnerability-scanning).

### Not included

- Authentication, encryption of the live database, a web or desktop interface, a full-screen terminal interface, published releases, and passive discovery. See [docs/ROADMAP.md](docs/ROADMAP.md).
