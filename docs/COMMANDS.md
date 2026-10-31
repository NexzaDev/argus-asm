# Command reference

This page lists every command, subcommand, and flag as implemented in `0.1.0-dev`. `argus-asm <command> -h` prints the same flags for one command. Flags may appear before or after positional arguments.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Unexpected failure, such as an unreadable database file or an I/O error |
| 2 | Usage error: unknown command or subcommand, missing or extra arguments, bad flag |
| 3 | Invalid input, not found, or already registered. Also used for failed imports and for a broken audit chain |
| 4 | Database not initialized. Run `init` |
| 5 | Target outside every authorized scope |
| 6 | Permission denied: unknown actor, missing actor, or role too low. Also used when active probing is not allowed for an asset |

## Common flags

Most commands accept:

- `--db PATH`: database file. Overrides `ARGUS_ASM_DB`. See [CONFIGURATION.md](CONFIGURATION.md).
- `--actor NAME`: the acting user. Overrides `ARGUS_ASM_ACTOR`. Required once any user exists. See [USAGE.md](USAGE.md#roles-and-actors).
- `--json`: machine-readable output on commands that list data.

`version`, `about`, `help`, and `diagnostics` do not need an actor.

## Setup

### `init`

Creates the database file and its parent directory if needed, applies pending migrations, and prints the schema version. Safe to run more than once. Accepts `--db`.

### `diagnostics`

Prints the app version, platform, database config source, status, schema version, and row counts. The database path is hidden unless `--show-path` is given. Accepts `--db`, `--show-path`, `--json`. Does not need an actor.

### `version`, `about`, `help`

`version` prints the version. `about [--json]` prints license, project URL, bug-report URL, commit, and build date. Build values show `unknown` unless the binary is built with the Makefile, which injects the version.

## Scope and inventory

### `scope add <domain|cidr>`

Authorizes a domain (including its subdomains) or a CIDR range. Analyst and viewer roles cannot run it. IPv4 CIDR ranges must have a prefix of `/16` or narrower (for example `/24`), and IPv6 ranges must have a prefix of `/32` or narrower. Broader ranges are refused as too broad. Requires the admin role.

### `scope list [--json]`

Lists authorized scopes. Requires the viewer role or higher.

### `asset add <domain|ip>`

Registers an asset. The value must fall inside an existing scope, otherwise the command exits 5. CIDR input is refused with a message that points to `scope add`. IPv4-mapped IPv6 addresses are stored in their IPv4 form. Requires the analyst role.

### `asset list [--json]`

Lists assets with owner, verification, probe permission, and creation time.

### `asset show <domain|ip> [--json]`

Shows one asset with owner and verification details.

### `asset remove <domain|ip> [--dry-run]`

Deletes an asset and cascades to its findings, services, certificates, relations, and tickets. `--dry-run` reports the counts without deleting. Requires the admin role. Each real removal is recorded in the audit log with the counts.

### `asset owner <domain|ip> --owner NAME [--verified-by NAME]`

Records the owning team or person. `--verified-by` records who confirmed ownership and sets the verification time to now. This is a record kept by the operator. The tool does not check ownership against any external source. Requires the analyst role.

### `asset link <from> <to> --kind resolves_to|hosts|related_to`

Records a relation between two registered assets. An asset cannot be linked to itself, and each (from, to, kind) combination can be recorded once. Requires the analyst role.

### `asset links <domain|ip> [--json]`

Lists relations in which the asset appears on either side.

### `asset probe allow|deny <domain|ip>`

Sets whether active TCP probing is permitted for one asset. Probing is denied by default. Requires the admin role. See [`probe run`](#probe-run-domainip---ports-list---execute---timeout-2s---concurrency-4).

## Services and certificates

### `service add <domain|ip> <port[/tcp|udp]> [--name NAME]`

Records an open service by hand. The protocol defaults to `tcp`. The name is up to 128 characters. Source is recorded as `manual`. Requires the analyst role.

### `service list [--asset ASSET] [--json]`

Lists recorded services, optionally for one asset. Requires the viewer role.

### `service import-nmap <file|-> [--dry-run]`

Imports open ports from Nmap XML output (produced with `-oX`). Only hosts with state `up` and ports with state `open` are imported. A host matches an inventory asset if one of its IPv4/IPv6 addresses or host names is registered. If any host does not match an asset, the whole import is refused and nothing is written. The file must not contain an internal DTD subset or entity declarations; a plain `<!DOCTYPE nmaprun>` line is accepted. Limits: 5 MiB, 10,000 hosts, and 10,000 open-port services in total. Requires the analyst role.

### `cert import <domain|ip> <pem-file> [--dry-run]`

Imports every `CERTIFICATE` block in a PEM file for an asset. Stored fields are subject, issuer, expiry, and SHA-256 fingerprint. Blocks that are not certificates, including private keys, are ignored and never stored. Certificates already known for the asset (same fingerprint) are counted as duplicates. Certificates that expire within 30 days create a notification. Limits: 5 MiB and 100 certificates per file. Requires the analyst role.

### `cert list [--json]`

Lists certificates ordered by expiry. Requires the viewer role.

## Active checks

### `probe run <domain|ip> --ports LIST [--execute] [--timeout 2s] [--concurrency 4]`

Plans or runs a TCP connect check against one asset.

- Without `--execute` the command prints the plan and contacts nothing. The asset must still be allowed for probing.
- With `--execute` it opens one TCP connection per port. A port is reported open only when the handshake completes. Any other outcome is reported as closed or filtered.
- `LIST` is a comma-separated set of ports from 1 to 65535, at most 64 distinct values.
- `--timeout` is between 100ms and 10s (default 2s). `--concurrency` is between 1 and 16 (default 4).
- The asset must be inside a scope, and an administrator must have run `asset probe allow` for it. Otherwise the command exits 6 (not allowed) or 5 (out of scope).
- Open ports are stored as services with source `tcp-connect`. The run is recorded in the audit log.

Domain assets are resolved by the operating system resolver when probed. No other probe types exist.

## Local lab

### `lab serve [--ports 18080,18022]`

Listens on `127.0.0.1` only, on unprivileged ports (1024 to 65535, at most 8). Each connection receives one synthetic banner line and is closed. The lab runs until Ctrl-C. It is intended as a loopback target for trying `probe run` safely. It does not need a database.

## Findings

### `findings import --format csv|json|sarif --source NAME <file|-> [--dry-run]`

Imports findings. `--source` is required and labels the batch. `--format` defaults to `csv`.

- CSV needs the header `asset,title,severity,source`. Each row's `source` must be filled in.
- JSON is an array of objects with the keys `asset`, `title`, `severity`, and `source`. Unknown keys are refused.
- SARIF 2.1.0 (`--format sarif`) reads every result in every run. The asset is the host or IP address in the first physical location's URI, and it must already be in the inventory. The title is the rule ID, or the message text when there is no rule ID. Severity comes from `properties.security-severity` when present (9.0 and above critical, 7.0 and above high, 4.0 and above medium, above 0 low, otherwise info). Without it, `level` maps `error` to high, `warning` or no level to medium, `note` to low, and `none` to info. The source column is the tool's `driver.name`. A result with no location, or with an unknown level, is a row error. The file must declare version exactly `2.1.0`. Limits: 5 MiB and 10,000 results.
- Severity is one of `critical`, `high`, `medium`, `low`, `info`.
- Every asset must already be in the inventory. Any invalid row refuses the whole file with `line N: message` on standard error and exit 3. Nothing is written.
- `--dry-run` validates the file and writes nothing.
- Limits: 5 MiB, 10,000 rows, 512 characters per field.
- A finding is identified by asset, title (case-insensitive), and source. Re-importing updates its severity and last-seen time.
- Critical and high findings create a notification (see `notify refresh`).

Requires the analyst role.

### `findings list [--status S] [--severity V] [--asset TEXT] [--all] [--limit N] [--explain] [--json]`

Lists findings ordered by risk score, highest first. By default, resolved findings are hidden. Accepted findings are shown with score 0. `--all` includes resolved findings. `--status` accepts `open`, `accepted`, or `resolved`. `--asset` is a substring match. `--explain` prints each score factor under its row. Requires the viewer role.

### `findings show <id> [--json]`

Shows one finding with its risk factors, linked tickets, and the services recorded for its asset.

### `findings status <id> --set open|accepted|resolved [--reason TEXT]`

Changes a finding's status. Accepting a risk requires `--reason` (up to 500 characters). Setting the status it already has is refused. Requires the analyst role.

### `findings batches [--source NAME] [--json]`

Lists import batches, newest first, with counts.

### `findings changes --source NAME [--json]`

Compares the two most recent imports from one source. It reports findings that are new, findings missing from the latest import, and findings whose severity changed. `--source` is required because comparing imports from different sources would not be meaningful. At least two imports from that source are needed.

## Risk score

The score is explainable: every factor appears under `findings list --explain`.

| Factor | Contribution |
|---|---|
| Severity | critical 90, high 70, medium 40, low 15, info 5 |
| Exposure | 2 per open recorded service on the asset, counting at most 5 services (10 points) |
| Age | +5 when an open finding was first seen 30 or more days ago |

The total is capped at 100. Accepted and resolved findings score 0.

## Tickets

### `ticket create --finding ID --owner NAME --due YYYY-MM-DD`

Creates a remediation ticket for a finding. The owner is up to 64 characters. Requires the analyst role.

### `ticket list [--overdue] [--json]`

Lists tickets ordered by due date. `--overdue` shows tickets that are not done and whose due date is before today (UTC).

### `ticket status <id> --set open|in_progress|done`

Changes a ticket's status. Setting the status it already has is refused. Requires the analyst role.

## Search, notifications, and export

### `search <term> [--json]`

Case-insensitive substring search over asset values and owners, finding titles, and service names. The term must be at least 2 characters.

### `notify list [--all] [--json]`

Lists notifications. Acknowledged ones are hidden unless `--all` is given.

### `notify ack <id>`

Acknowledges a notification. Requires the analyst role.

### `notify refresh`

Creates notifications for overdue tickets and certificates that expire within 30 days. Running it again creates nothing new. Requires the analyst role.

Notifications are stored in the database and shown by `notify list`.

### `notify deliver --to FILE [--dry-run]`

Appends every undelivered notification to a local file as one JSON object per line: `id`, `created_at`, `kind`, and `message`. Requires the analyst role.

- `--to` is required. It must be a regular file or a path that does not exist yet. A symbolic link is refused. A new file is created with mode `0600` on Unix-like systems.
- `--dry-run` reports the number of undelivered notifications and writes nothing.
- When nothing is pending, the command reports `delivered 0 notification(s)` and creates no file.
- Delivery is at-least-once. The lines are written first, and only then are the notifications marked delivered. If the process stops between the two steps, the next run writes those lines again. Use the `id` field to ignore duplicates.
- The command makes no network connection, runs no hook, and sends no email, chat message, or webhook. Forwarding the file anywhere else is up to you.

### Background notifier: `notifier run|install|uninstall|status`

An opt-in, per-user process that shows an OS notification when new notifications are stored. It is off by default and nothing starts unless you run `notifier install --yes`.

- `notifier run [--db PATH] [--interval 5m] [--detail minimal|detailed]` opens the database read-only (SQLite `mode=ro`) and checks it on the interval. The interval must be between `1m` and `24h`.
- `minimal` (default) shows counts only, for example `2 new critical findings in Argus`. `detailed` also shows up to five stored notification messages, each cut to 120 characters. Evidence and credentials are never included.
- `notifier install --yes [--db PATH] [--interval ..] [--detail ..]` writes a per-user autostart entry and records a baseline, so only notifications created after installation are announced. Without `--yes` the command prints what it would do and exits with a usage error. It needs no administrator rights.
  - Windows: `Startup` folder, `argus-asm-notifier.cmd`.
  - macOS: `~/Library/LaunchAgents/com.nexzadev.argus-asm.notifier.plist`.
  - Linux: `~/.config/autostart/argus-asm-notifier.desktop` (XDG autostart).
- `notifier uninstall` removes the autostart entry only. It keeps the stored baseline file.
- `notifier status` reports whether the autostart entry exists, where the baseline is, and whether the OS notification tool is on `PATH`.
- OS notification tools: `notify-send` (Linux), `osascript` (macOS), and PowerShell with a balloon tip (Windows). The message travels as a process argument or environment variable, never through a shell. The notifier makes no network connection.
- On Windows the program starts PowerShell with `-NoProfile -NonInteractive -ExecutionPolicy Bypass -Command` and one fixed script. The program sets the execution-policy flag; users cannot change it. The Windows path has not been run on a real desktop.
- Real OS display of these notifications is not proven by the automated tests. See `TESTING.md`.

### `export findings|assets --format csv|json [--all] [--output FILE]`

Exports findings or assets. Findings exclude resolved items unless `--all` is given. CSV cells that begin with `=`, `+`, `-`, `@`, tab, or carriage return are prefixed with an apostrophe so that spreadsheet programs do not run them as formulas. `--output` writes to a file created with mode `0600` on Unix-like systems. Requires the analyst role.

## Users and audit

### `user add <name> --role viewer|analyst|admin`

Adds a user. Names are 1 to 64 letters, digits, `.`, `_`, or `-`. The name `local` is reserved. While no users exist, only the first user may be created, and it must have the admin role. Requires the admin role once users exist.

### `user list [--json]`

Lists users. Requires the viewer role.

### `user role <name> --role viewer|analyst|admin`

Changes a user's role. The last administrator cannot be demoted. Requires the admin role.

### `audit list [--json]`

Lists the audit log. Requires the admin role.

### `audit verify [--json]`

Recomputes the hash chain and reports how many entries are chained and how many are legacy entries from before chaining was introduced. Exits 3 if the chain is broken. Requires the admin role. See [THREAT_MODEL.md](THREAT_MODEL.md) for what this does and does not prove.

## Sealed copies

The live database is plaintext while it is open. These commands make an encrypted copy of it and turn a copy back into a database file. See [SECURITY.md](SECURITY.md#encrypted-copies) for the model and its limits.

### `db seal --out FILE --passphrase-stdin [--db PATH] [--actor NAME]`

Writes an encrypted snapshot of the database to `FILE`. Requires the administrator role.

- The passphrase is read from the first line of standard input. It is never read from an argument or an environment variable. It must be at least 12 bytes.
- `FILE` must not exist. It is created with mode `0600` on Unix-like systems.
- The snapshot is taken with SQLite `VACUUM INTO` into a temporary file in the database's directory. That temporary file holds plaintext for a moment, is created with mode `0600`, and is removed before the command returns.
- The command records a `db.snapshot` entry in the audit log.
- A refused seal (wrong role, short passphrase, existing file) leaves no file behind.

### `db unseal --in FILE --out FILE --passphrase-stdin`

Decrypts a sealed file into a new database file. This command needs no actor and no existing database, because it only creates the output file.

- The passphrase is read the same way as for `db seal`.
- `--out` must not exist. It is created with mode `0600` on Unix-like systems.
- A wrong passphrase, or a damaged or altered file, exits 3 and writes nothing.
- After decrypting, the command opens the result and checks that it is an argus-asm database. If that check fails, the output is removed.
- The unsealed file keeps its schema version. Run `init` against it to apply newer migrations.

`db` is not available inside `console`, because the passphrase is read from standard input and the console reads its commands from the same stream. Run `db` from the shell.

## Console

### `console [--db PATH] [--actor NAME]`

Starts a line-based shell. Each line is one command from this reference, for example `findings list --explain`. The `--db` and `--actor` values given to `console` apply to each command that accepts them, unless the line sets its own.

- Blank lines and lines starting with `#` are ignored. Quotes group words (`"like this"` or `'like this'`).
- `help` prints the usage summary. `exit` or `quit` leaves the console.
- A failing command prints its error and the console continues.
- Inside the console, `-` (stdin) is not available, so read input from a file path.
- The console cannot be nested.

The console is not a full-screen terminal interface. It reads plain lines, so it works the same on every platform and in scripts that pipe commands in.

## Launcher and preferences

A bare `argus-asm` (no command) starts the launcher. Its behavior depends on the UI mode, which is resolved in this order: `--ui`, then `ARGUS_UI`, then the stored preference (`config set-ui`), then nothing set.

- `gui`: opens the dashboard window. Needs a graphical display and a build with the GUI (the default build).
- `console`: starts the terminal console (see the Console section).
- `cli`: prints usage and exits with code 2.
- Not set: on a desktop the launcher shows a chooser with **Dashboard (graphical window)**, **Terminal**, and **Remember my choice**. Without a display, or in a `nogui` build, it starts the terminal console and prints the reason to standard error.

When `gui` is requested but no display is available (for example over SSH, or on Linux with `DISPLAY` and `WAYLAND_DISPLAY` unset), the launcher prints the reason and starts the console instead.

### `--ui gui|console|cli` (with no command)

Chooses the launcher mode for one run. `--ui` cannot be combined with a command; for example `argus-asm --ui console init` exits with code 2.

```sh
argus-asm --ui gui
ARGUS_UI=console argus-asm
```

### `config show`

Prints the resolved UI mode and its source (`flag`, `environment`, `stored`), or `not set`.

### `config set-ui gui|console|cli`

Stores the UI preference in `ui.conf` inside the per-user configuration directory (`argus-asm/ui.conf`, mode 0600). The file contains one line, for example `ui=gui`.

### `config reset-ui`

Removes the stored preference, so the launcher asks again on a desktop. Running it again reports that there is nothing to reset. If `ui.conf` holds an unknown value, other commands exit with code 2 and point to this command.
