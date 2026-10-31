# argus-asm

A local, defensive tool with a command line, an interactive console, and a desktop GUI for recording the assets you are authorized to assess, importing findings, tracking remediation, and reviewing risk. Everything runs on your computer, in one SQLite file. It needs no account, server, or paid service.

> **Status: early development (`0.1.0-dev`).** The features below are implemented and covered by tests. Verified platforms are listed in [docs/PLATFORM_SUPPORT.md](docs/PLATFORM_SUPPORT.md). Not every feature in the original specification exists yet. See [docs/ROADMAP.md](docs/ROADMAP.md).

## What it does

- **Scope first.** Declare a domain (covering its subdomains) or a CIDR range you are authorized to assess. Assets outside every scope are refused.
- **Inventory.** Assets, services (manual entry, Nmap XML import, or probe results), certificates (PEM import), and links between assets.
- **Findings.** Import from CSV, JSON, or SARIF 2.1.0, all-or-nothing, with `--dry-run`. Findings are deduplicated. Each one has an explainable risk score.
- **Lifecycle.** Open, accepted (with a required reason), and resolved states. Remediation tickets with owners and due dates. Change detection between two imports from one source.
- **Active checks, off by default.** TCP connect checks only, on assets in scope that an administrator has allowed, and only with `--execute`. A loopback lab lets you try them safely.
- **Notifications, search, export.** Notifications are stored in the database, and `notify deliver` appends them to a local file you choose. Search covers assets, findings, and services. Export writes CSV or JSON, with formula neutralization.
- **Roles and audit.** Viewer, analyst, and administrator roles, and an append-only, hash-chained audit log.
- **Console.** A line-based shell for a session with many commands.
- **Desktop GUI.** A native window (Fyne) with a sidebar, tables, an Overview, notifications, and an About page with the usage guide. Running `argus-asm` with no command opens a launcher that asks whether to start the dashboard or the terminal. The preference can be remembered (`config set-ui`, `config reset-ui`, `--ui gui|console|cli`, `ARGUS_UI`).
- **Background notifier (opt-in).** `argus-asm notifier install` registers a per-user autostart entry that reads the local database read-only and shows operating-system notifications. It is off by default. Real notification display is not verified on every operating system.

## What it does not do

- It does not discover hosts, scan ports beyond the allowed TCP connect checks, or contact any host you have not allowed.
- It does not verify ownership or authorization against any outside source. Scope and ownership records are declarations that you make.
- It does not authenticate users. The actor name is a label, not a login. Do not treat roles as a security boundary on a shared machine. See [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md).
- It does not encrypt the live database. `db seal` writes an encrypted copy, which you can store and restore with `db unseal`.
- It does not send notifications over a network. `notify deliver` writes them to a local file, and nothing else reads it.
- It does not provide a web interface or a full-screen terminal interface.

## Download

The [Releases page](https://github.com/NexzaDev/argus-asm/releases) has pre-release builds for Windows (x64), macOS (Apple silicon, arm64), and Linux (x64). They are built by GitHub Actions, are unsigned, and were not tested by hand on real desktops.

- **macOS:** extract the archive, right-click `argus-asm` and choose Open the first time.
- **Windows:** if SmartScreen appears, choose More info, then Run anyway.
- **Linux:** extract the archive and run `chmod +x argus-asm`.

## Supported platforms

| Platform | Status |
|---|---|
| Linux amd64 | Built and tested locally and in CI |
| macOS (`macos-latest` runner) | Tested in CI |
| Windows amd64 (`windows-latest` runner) | Tested in CI |

Details: [docs/PLATFORM_SUPPORT.md](docs/PLATFORM_SUPPORT.md).

## Prerequisites

- Go 1.27.3 (declared in `go.mod`). Earlier Go versions have reachable standard-library vulnerabilities; see [docs/SECURITY.md](docs/SECURITY.md#vulnerability-scanning).
- A C compiler, because the SQLite driver uses cgo: `gcc` on Linux, Xcode Command Line Tools on macOS, MinGW-w64 GCC on Windows
- On Linux, building the GUI also needs the graphics development packages: `libgl1-mesa-dev`, `xorg-dev`, `libwayland-dev`, `libxkbcommon-dev`. `go build -tags nogui` builds without the GUI and without them
- GNU Make (optional; every target has a plain `go` equivalent)

## Quick start

```sh
git clone https://github.com/NexzaDev/argus-asm.git
cd argus-asm
make build                      # or: go build -trimpath -o bin/argus-asm ./cmd/argus-asm

export ARGUS_ASM_DB="$PWD/build/demo/quick.db"
./bin/argus-asm init
./bin/argus-asm scope add example.test
./bin/argus-asm scope add 192.0.2.0/24
./bin/argus-asm asset add www.example.test
./bin/argus-asm asset add api.example.test
./bin/argus-asm asset add 192.0.2.10
./bin/argus-asm findings import --source demo-scanner testdata/demo/findings.csv --dry-run
./bin/argus-asm findings import --source demo-scanner testdata/demo/findings.csv
./bin/argus-asm findings list --explain
```

On Windows PowerShell, build with `go build -trimpath -o bin\argus-asm.exe .\cmd\argus-asm`, use `.\bin\argus-asm.exe`, and set `$env:ARGUS_ASM_DB` instead of `export`.

`make demo` runs a complete labeled demo into `build/demo/argus-asm-demo.db`. The walkthrough with every step is in [docs/USAGE.md](docs/USAGE.md).

## Commands

| Area | Commands |
|---|---|
| Setup | `init`, `diagnostics`, `version`, `about`, `help` |
| Scope and inventory | `scope add\|list`, `asset add\|list\|show\|remove\|owner\|link\|links\|probe`, `service add\|list\|import-nmap`, `cert import\|list` |
| Active checks | `probe run` (plan by default; `--execute` to connect), `lab serve` |
| Findings | `findings import\|list\|show\|status\|batches\|changes` |
| Tickets and notifications | `ticket create\|list\|status`, `notify list\|ack\|refresh` |
| Search and export | `search`, `export findings\|assets` |
| Access and audit | `user add\|list\|role`, `audit list\|verify` |
| Interactive and desktop    | `console`, `config show|set-ui|reset-ui`, `notifier run|install|uninstall|status`, `--ui gui|console|cli` |

Every command and flag: [docs/COMMANDS.md](docs/COMMANDS.md).

## Configuration

The database location comes from `--db`, then the `ARGUS_ASM_DB` environment variable, then a per-user default. The acting user comes from `--actor` or `ARGUS_ASM_ACTOR`. See [docs/CONFIGURATION.md](docs/CONFIGURATION.md). `.env.example` lists the variables with placeholder values. The program does not read `.env` files.

## Testing and checks

```sh
make check        # gofmt check, go vet, tests, build
make race         # tests with the race detector
make fuzz-smoke   # 10-second fuzz run for each importer parser
make demo         # labeled synthetic demo database
```

Without `make`, the plain commands are in [docs/TESTING.md](docs/TESTING.md). Continuous integration (on pull requests and pushes to `main`) runs formatting, vet, race-enabled tests, a build, and a smoke test on Linux, macOS, and Windows, runs staticcheck 2026.2.2 and govulncheck v1.9.0 with Go 1.27.3, and builds a per-platform archive with a SHA-256 file as a workflow artifact. Pushing a tag that starts with `v` runs the Beta release workflow, which builds the Linux, macOS arm64, and Windows packages and attaches them to a pre-release. See [docs/TESTING.md](docs/TESTING.md#continuous-integration) for the job list and the commands.

## Security notes

- No network calls except the optional `probe run --execute`, which contacts only allowed assets in scope. No telemetry.
- The lab listens on `127.0.0.1` only, while it runs.
- The database is created with owner-only permissions on Unix. The live database is not encrypted. Sealed copies (`db seal`) are, and the model is described in [docs/SECURITY.md](docs/SECURITY.md#encrypted-copies).
- Imports are size- and row-limited, validated before writing, and written in one transaction.

Read [docs/SECURITY.md](docs/SECURITY.md) and [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) before using it with real data. To report a vulnerability, follow [SECURITY.md](SECURITY.md).

## Documentation

- [Documentation index](docs/README.md)
- [Installation](docs/INSTALLATION.md) · [Usage](docs/USAGE.md) · [Commands](docs/COMMANDS.md) · [Configuration](docs/CONFIGURATION.md)
- [Architecture](docs/ARCHITECTURE.md) · [Testing](docs/TESTING.md) · [Platform support](docs/PLATFORM_SUPPORT.md)
- [Security](docs/SECURITY.md) · [Threat model](docs/THREAT_MODEL.md) · [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Contributing](CONTRIBUTING.md) · [Roadmap](docs/ROADMAP.md) · [Changelog](CHANGELOG.md)
- [Original product specification](docs/SPECIFICATION.md)

## License

MIT. See [LICENSE](LICENSE) and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). The copyright holder in `LICENSE` is NexzaDev, as confirmed by the maintainer.
