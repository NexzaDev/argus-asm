# Roadmap and status

This page tracks the product requirements from [SPECIFICATION.md](SPECIFICATION.md) against the code. Status words are exact:

- **Implemented**: built, documented in [COMMANDS.md](COMMANDS.md) or [USAGE.md](USAGE.md), and covered by tests listed in [TESTING.md](TESTING.md).
- **Partial**: some of the requirement is built. The gap is listed.
- **Not implemented**: not built in this release. Where a reason is given, it says why it is deferred.

## Core workflow

| Requirement | Status | Notes |
|---|---|---|
| Scope-first authorization (domain and CIDR scopes, refusal outside scope) | Implemented | IPv4 `/16` or narrower, IPv6 `/32` or narrower |
| Asset inventory (domains, IPs) | Implemented | IPv4-mapped IPv6 normalized to IPv4 |
| Verified ownership records | Partial | Owner and verifier are recorded by the operator. Nothing is checked against an external source. |
| Service inventory | Implemented | Manual entry, Nmap XML import, and TCP connect results |
| Certificate inventory | Implemented | PEM import with fingerprint, subject, issuer, and expiry |
| Asset relationships | Implemented | `resolves_to`, `hosts`, `related_to` |
| Deduplication | Implemented | Findings by asset kind and value, title, and the row's source column (the `--source` flag only labels the batch). Assets by kind and value. |
| Passive discovery | Not implemented | Deferred. Certificate-transparency and DNS-history lookups depend on third-party services with their own terms and rate limits, and they would add outbound network calls, which the program does not make today. Both need a source decision and a threat-model review first. |
| Dry-run for disruptive operations | Implemented | `--dry-run` on imports, asset removal, and notification delivery. Probing plans without `--execute`. |

## Importers

| Format | Status |
|---|---|
| Findings CSV | Implemented |
| Findings JSON array | Implemented |
| SARIF 2.1.0 (`findings import --format sarif`) | Implemented. Synthetic fixture in `testdata/demo/findings.sarif`. |
| Nmap XML (`-oX`) | Implemented, open ports only |
| PEM certificates | Implemented |
| Other scanner formats (vendor exports and others) | Not implemented |

## Risk and lifecycle

| Requirement | Status | Notes |
|---|---|---|
| Explainable risk score | Implemented | Factors shown with `findings list --explain` and `findings show` |
| Finding lifecycle (open, accepted, resolved) | Implemented | Accepting requires a reason |
| Remediation tickets | Implemented | Create, list, overdue filter, status |
| Change detection | Implemented | Compares the two latest imports from one source: new, missing, severity changed |

## Active checks

| Requirement | Status | Notes |
|---|---|---|
| Off by default | Implemented | |
| Allowlisted assets, scope check | Implemented | Per-asset permission set by an administrator |
| Bounded ports and concurrency | Implemented | 64 ports, 16 workers, bounded timeouts |
| Protocol checks beyond TCP connect | Not implemented | No UDP, no banner grabbing, no TLS inspection |

## Notifications, search, export

| Requirement | Status | Notes |
|---|---|---|
| Notifications | Implemented (M4) | Stored in the database and listed with `notify list`. `notify deliver --to FILE` appends undelivered ones to a local file (at-least-once, mode `0600`, no network). The GUI Notifications category has an unread bell, severity filter, and Mark as read. Opt-in `notifier run\|install\|uninstall\|status` shows OS notifications via per-user autostart, reading the database read-only. Real OS display is not proven by automated tests. Email, chat, and webhook delivery are not implemented, because each one needs network code and a threat-model review. |
| Search | Partial | Case-insensitive substring over assets, owners, finding titles, and service names. No query language or indexing. |
| Exports | Implemented | CSV and JSON for findings and assets, with formula neutralization |

## Storage and encryption

| Requirement | Status | Notes |
|---|---|---|
| Encryption at rest | Partial | `db seal` and `db unseal` make AES-256-GCM copies under a passphrase, using the standard library only. The live database stays plaintext while it is open. See [SECURITY.md](SECURITY.md#encrypted-copies). |
| Key management model | Implemented | The passphrase is the only key and is read from standard input. It is never stored. There is no recovery and no rotation in place. Documented in [SECURITY.md](SECURITY.md#key-management). |
| Schema migrations | Implemented | Versions 1 to 4, applied by `init`. Migration tests cover an upgrade from schema 2. |

## Access and audit

| Requirement | Status | Notes |
|---|---|---|
| RBAC (viewer, analyst, administrator) | Partial | Roles are enforced in the service layer. The actor is a name, not an authenticated identity. |
| Authentication | Not implemented | See [THREAT_MODEL.md](THREAT_MODEL.md). |
| Audit log | Implemented | Append-only at the database level, hash-chained, verifiable by administrators |

## Interfaces

| Requirement | Status | Notes |
|---|---|---|
| CLI with consistent commands, help, and exit codes | Implemented | |
| Line-based console | Implemented | Same commands, plain lines. Not a full-screen interface. `db` is refused inside the console. |
| Full-screen TUI | Not implemented | Deferred. A full-screen interface needs raw terminal mode on each OS, which cannot be tested on macOS or Windows from this environment, and any terminal library needs maintainer approval. [ADR 0004](adr/0004-line-based-console.md) still applies; round 4 did not change that decision. |
| Launcher (`--ui`, `ARGUS_UI`, `config`) | Implemented | Modes `gui`, `console`, `cli`. Unset on a desktop shows a chooser; without a display, the console starts and the reason is printed. See [ADR 0005](adr/0005-desktop-gui-and-launcher.md). |
| Desktop GUI (Fyne) | Implemented in round 6 code; CI not run | Milestones 1 to 5: launcher and chooser, sidebar dashboard with the twelve categories, tables, Overview, About, Notifications bell and filter, the icon, the opt-in notifier, and a manual release workflow. On this branch, CI run 40918812174 (commit `0055668b`) passed the Ubuntu and Windows test, GUI, and smoke jobs, plus the design screenshots. The macOS jobs never got a runner. Later runs, including the run for the current HEAD, started no steps. See [TESTING.md](TESTING.md#ci-on-this-branch-checked-from-run-metadata-not-logs). Real windows on Windows, macOS, and Linux are not verified. The round-4 reasons for deferring the GUI are kept in the section below. |
| Offline usage guide | Implemented as documentation | `docs/USAGE.md` and `argus-asm help` work without network access. A separate searchable in-binary guide was not built. |
| Web or HTTP interface | Not planned | Out of scope by design. |
| Local lab for safe testing | Implemented | `lab serve`, loopback only |

## Platform and release

| Requirement | Status | Notes |
|---|---|---|
| Linux build and tests | Implemented and verified locally and in CI on Go 1.27.3 | See [TESTING.md](TESTING.md#recorded-results). |
| macOS and Windows build and tests | Verified in CI on Go 1.27.3 (run 40821785897, earlier branch; see PLATFORM_SUPPORT.md) | Not run on local machines. See [PLATFORM_SUPPORT.md](PLATFORM_SUPPORT.md). |
| Static analysis (staticcheck 2026.2.2) in CI | Verified in CI (run 40821785897, earlier branch; see PLATFORM_SUPPORT.md) | Runs on the `go.mod` toolchain. 2025.1.1 cannot read Go 1.27 export data, so it was replaced. |
| Vulnerability scanning (govulncheck v1.9.0 on Go 1.27.3) in CI | Verified in CI (run 40821785897, earlier branch; see PLATFORM_SUPPORT.md) | govulncheck reports no vulnerabilities on Go 1.27.3. See [SECURITY.md](SECURITY.md#vulnerability-scanning). |
| Test jobs on the exact Go version in `go.mod` (1.27.3) | Verified in CI (run 40821785897, earlier branch; see PLATFORM_SUPPORT.md) | The separate `minimum-go` job was removed. The test matrix already runs the exact version. |
| Go line | Decided: Go 1.27.3 | Set in `go.mod`, CI, and docs. See the decisions below. |
| Release artifacts and checksums | Implemented, not run | `release.yml` (manual, `workflow_dispatch`) builds the GUI with `fyne package` for Linux, macOS, and Windows, writes `SHA256SUMS`, and creates a draft GitHub Release. `ci.yml` `release-artifacts` builds the CLI archive per runner. `release.yml` has never run on GitHub. The `release-artifacts` job in `ci.yml` ran on commit `0055668b`: the Ubuntu and Windows steps passed, and the macOS job never got a runner. Nothing is signed. |
| Release signatures | Not implemented | Deferred until a release is published. A signing key needs a decision and a secure home. |
| Structured logs and health endpoints | Not implemented | The CLI writes plain text. `diagnostics` is the health report. |
| Dependency and license inventory | Partial | `THIRD_PARTY_NOTICES.md` lists the one runtime dependency. A generated SBOM is not produced. |
| Private vulnerability reporting | Enabled at publication | Reporters use the advisory form on the Security tab. Email is the backup channel. See [SECURITY.md](../SECURITY.md). The maintainer confirms that the Security tab shows Report a vulnerability. |

## Pre-public checklist

Before the repository is made public, every item below must be checked. An item is checked only when its check has run.

- [x] **Secrets in the tree.** Checked 2026-10-29 with a pattern scan for GitHub, AWS, Slack, and Google tokens, private keys, and `password`, `secret`, `token`, or `api_key` assignments. No matches.
- [x] **Secrets in history.** Checked 2026-10-29 with the same patterns over all added lines in `git log --all -p`. No matches.
- [ ] **Personal data in commit metadata.** Checked 2026-10-29. Commit `c0391ab` (the base commit of `main`, authored by `Nexza`) has a personal email address in its author field. The tree itself contains no email addresses. Removing it from history would rewrite `main`, so it needs a maintainer decision. See the decisions below.
- [x] **Synthetic demo data.** Names use RFC 2606 domains (`example.test`) and addresses use RFC 5737 ranges. The demo certificate is self-signed for `www.example.test`. Examples that used `8.8.8.8` now use `203.0.113.99`.
- [x] **Environment files.** `.env.example` contains placeholders only. The CLI reads environment variables, not `.env` files, as stated in [CONFIGURATION.md](CONFIGURATION.md).
- [x] **Ignored artifacts.** `.gitignore` covers `bin/`, `build/`, `dist/`, `*.db` and its journals, `*.sealed`, `.argus-asm-snapshot-*.tmp`, and `.env`.
- [x] **License consistency.** `LICENSE` (MIT, Copyright (c) 2026 NexzaDev), the README, and `THIRD_PARTY_NOTICES.md` agree. The one runtime dependency, `github.com/mattn/go-sqlite3` v1.14.52, is MIT (Copyright (c) 2014 Yasuhiro Matsumoto), as its module LICENSE states.
- [x] **Security reporting address.** The placeholder is replaced: the advisory form is the preferred channel and a backup email is listed in [SECURITY.md](../SECURITY.md).
- [x] **Private vulnerability reporting.** Enabled at publication. Confirm the Security tab shows Report a vulnerability.
- [ ] **Release artifact job.** The `release-artifacts` CI job has run successfully on Linux, macOS, and Windows, and the artifacts were downloaded and checked against their SHA-256 files.
- [ ] **Repository settings.** Branch protection on `main` and Actions permissions are set by a maintainer. Nothing in this repository configures them.
- [ ] **Docs against behavior.** Every walkthrough in [USAGE.md](USAGE.md) has been run on the release commit, and the output shown matches.

## Open decisions

These need a maintainer decision. Nothing below was changed without one, except the Go line, which the maintainer chose.

1. **Go line (decided).** The `go` directive is `1.27.3`, the current patch release of Go 1.27. It is used in `go.mod`, CI, and the docs. Go 1.25 is out of upstream support. Revisit this when a later Go patch or line is released.
2. **Private vulnerability reporting (decided).** Enabled at publication. Email is the backup channel.
3. **Security email address (decided).** The backup address is listed in [SECURITY.md](../SECURITY.md).
4. **Personal email in commit metadata (decided).** The repository was republished as a single commit authored with the GitHub noreply address.
5. **Multi-user access.** Not planned. Any work on it needs the review items in [THREAT_MODEL.md](THREAT_MODEL.md#security-implications-of-multi-user-access).
6. **Release signing.** Not decided. Signing would need a key the maintainer controls.

## Next steps

Candidates for the next milestone, in rough order:

1. Run the CI matrix on GitHub and fix any failures, including the `release-artifacts` job.
2. Tag a first release with checksums, after the pre-public checklist is complete.
3. Authentication for multi-user use, if that becomes a requirement. This needs a new threat-model review.
4. Additional importers from formats the maintainer can test with synthetic data.
5. Notification delivery to a network endpoint, if there is a demand for it. This needs a threat-model review, because it adds outbound network code.

## Fyne GUI build plan (status in round 6)

Status: milestones 1 to 5 are implemented in the repository as described in docs/adr/0005-desktop-gui-and-launcher.md. The rest of this section is the original plan. Where it says a step is not done, the milestone status below says what was done instead.

### Milestone status (round 6)

- M1 GUI foundation: implemented. CI jobs exist for Ubuntu, macOS, and Windows, but none has run on GitHub since this work started.
- M2 dashboard: implemented. Tooltips are replaced by the status line (Fyne has no tooltip widget here).
- M3 icon: implemented as an original eye-with-network-iris design in `assets/icon/`. The window icon, the About logo, and the chooser use the embedded 512 px PNG. Taskbar and dock appearance is not verified on any OS. Asset shapes are checked by `TestIconAssetsAreWellFormed`.
- M4 notifications: implemented. Bell, filter, mark-as-read, and `notifier run|install|uninstall|status`. Real OS notification display and login autostart are not verified. The notifier is tested with an injected sender.
- M5 releases: implemented, not run. `release.yml` is manual, builds with the pinned `fyne.io/tools/cmd/fyne@v1.7.4`, writes SHA-256 checksums, and creates a draft release. The packaging decision is in ADR 0005. The steps for publishing from a phone are in docs/RELEASING.md. Nothing is signed.
- Open for M5: the first GitHub run of `release.yml`; checking the artifacts on each OS; a manual run of each binary; `go-sum.yml` must be disabled on GitHub by the maintainer (HTTP 403 for the sandbox token). go.sum was regenerated and verified on a GitHub runner (run 40914349608, commit `a64dc88`) and checked locally (TESTING.md Round 7); the About screen shows the development version because `fyne package` does not set the build version variable.

The original plan follows.

Original status (round 4): nothing in this section existed in the repository at the time of writing. It records what a later milestone needs. It is to be carried out in an environment with normal network access (module proxy, GitHub, and the vanity hosts `fyne.io` and `golang.org`) and a display or a CI runner with GUI support. The round 4 sandbox had none of these.

### Scope

- A desktop window with the screens listed in the specification: Overview, Scope, Assets, Findings, Imports, Changes, Notifications, Access, Audit, Reports, Settings, About. Each screen calls the same service layer as the CLI. No business rules go into the GUI package.
- Each screen must have a working action or be left out. No placeholder controls.
- The GUI is local only. It opens no network listener.

### Dependencies and versions

- `fyne.io/fyne/v2 v2.8.2`, the latest release at the time of the round 4 check (`gh api repos/fyne-io/fyne/releases/latest`). Its `go.mod` declares `go 1.22.0`. The repository stays on `go 1.27.3`.
- Transitive modules that `fyne.io/fyne/v2 v2.8.2` requires, as listed in its `go.mod`: `fyne.io/systray` (pseudo-version `v1.12.3-0.20260810170012-af4e8e793ec4`), `golang.org/x/image v0.24.0`, `golang.org/x/mod v0.23.0`, `golang.org/x/sys v0.30.0`, `golang.org/x/text v0.22.0`, `golang.org/x/tools v0.30.0`, `golang.org/x/net v0.35.0` (indirect), `golang.org/x/sync v0.11.0` (indirect), and others. The final versions come from the module graph after `go mod tidy`, not from this list.
- No other GUI toolkit, no web view, and no browser runtime.
- Linux, macOS, and Windows need cgo and a C toolchain. Linux also needs the OpenGL and X11 development packages. The exact package list must be taken from Fyne's getting-started documentation for v2.8.2 when the work starts. It is not verified here.

### go.sum step

1. Add the `fyne.io/fyne/v2` require with `go get fyne.io/fyne/v2@v2.8.2` on a machine that can reach the proxy.
2. Run `go mod tidy` with `GOTOOLCHAIN=local` and `GOFLAGS` unset, so the module graph uses the repository's `go 1.27.3` line.
3. Commit `go.mod` and `go.sum` together. Confirm with `go mod verify` and with `make test` on each OS.
4. Do not add `replace` directives to `go.mod` for vanity hosts. Those were used only for a sandbox probe, not for the repository.
5. Record the resulting license of each new module in THIRD_PARTY_NOTICES.md and in the dependency inventory.

### Build tags

- The GUI package is `internal/gui`, with files that carry `//go:build !nogui`. A `nogui` stub file carries `//go:build nogui` and implements the same entry point, returning a usage error that says the GUI is not in this build.
- The `//go:build` lines are build constraints, not comments. The no-comments rule in the specification covers source comments, so these lines are the one exception that the build tool needs. This is a decision for the maintainer to confirm before the milestone starts.
- A default build includes the GUI. `go build -tags nogui ./cmd/argus-asm` excludes it.
- `nogui` excludes GUI code only. The CLI still needs cgo because the SQLite driver is cgo-based. A `nogui` build is not a cgo-free build.
- The `ci` tag is not used for the release build. Earlier sandbox checks compiled Fyne with `-tags ci` only to test the compile path. That tag disables the desktop driver and gives no desktop build.

### Launcher changes

- `internal/config/ui.go`: add `gui` to the valid modes. `ValidUIMode` depends on the build, so the set of valid modes comes from a build-tagged constant or function. `ParseStoredUI`, `FormatStoredUI`, and `ResolveUI` use that set.
- `ui.conf` keeps the same format (`ui=<mode>`). A file that stores `gui` in a `nogui` build is treated as invalid, and the error tells the user to run `argus-asm config reset-ui` or set another mode.
- `internal/cli/config.go`: `runLauncher` dispatches `gui` to the GUI entry point. Under `nogui`, `--ui gui`, `ARGUS_UI=gui`, and `config set-ui gui` each exit 2 with a clear message.
- `config set-ui` should reject `gui` in a `nogui` build with the same message.
- `--ui gui` remains valid only with no command, as today.
- The help text and `docs/COMMANDS.md` must list `gui` only when the build supports it. Docs are updated in the same change as the code.

### CI job layout

Existing jobs stay as they are. The plan adds jobs; it does not change the current ones.

- `test-gui-linux` (ubuntu-latest): installs the OpenGL and X11 packages confirmed in step 1, runs `go test ./...`, and runs the headless tests under `xvfb-run` (or the equivalent virtual display). Verifies `go build ./cmd/argus-asm` with and without `-tags nogui`.
- `test-gui-macos` (macos-latest): builds with and without `nogui`. Runs the Fyne test driver tests, which need no window. Whether a real window session is available on the hosted runner is not verified.
- `test-gui-windows` (windows-latest): same as macOS. Requires a C toolchain on the runner, which is not verified.
- Headless GUI tests use the Fyne test driver (`fyne.io/fyne/v2/test`) and must not depend on a display server. Tests that need a display are marked and run only where one exists.
- The `nogui` build is tested on all three OSes, so the CLI-only path is checked everywhere.
- The release artifacts job builds the default (GUI) binary on each OS only after these jobs pass. Each OS artifact is produced on its own runner.
- The minimum permissions and pinned action versions remain as they are in `ci.yml`.

### Verification required before claiming the GUI

- Each OS: `go build`, `go test`, and `go vet` pass with GUI enabled and with `nogui`.
- Each screen has a test that exercises its action through the service layer, using the test driver.
- A manual run on each OS, recorded in TESTING.md with the OS version, Go version, and exact commands. Until then, the platform is not claimed as supported.
- The About screen shows the repository URL from the central metadata location, or an honest fallback if it is not configured.

### Not decided

- Whether the default build includes the GUI, or whether the GUI is an opt-in build. The plan assumes included.
- Whether the `//go:build` exception to the no-comments rule is acceptable.
- Whether headless GUI tests can run on the hosted macOS and Windows runners. This must be tested, not assumed.
