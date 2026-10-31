# Configuration

argus-asm is configured with command-line flags, environment variables, and one stored launcher preference (`ui.conf`, below). It does not read any other configuration file, and it does not read `.env` files. `.env.example` lists the variables as placeholders so you can copy the names. Export them in your shell or set them in your process environment.

## Environment variables

| Variable | Purpose | Default |
|---|---|---|
| `ARGUS_ASM_DB` | Path to the SQLite database file | Per-user location (below) |
| `ARGUS_ASM_ACTOR` | Acting user name for commands that need one | none |
| `ARGUS_UI` | Launcher mode for a bare `argus-asm` run: `gui`, `console`, or `cli` | stored preference, then the chooser or console |

Command-line flags take precedence over environment variables. `--db` overrides `ARGUS_ASM_DB`, and `--actor` overrides `ARGUS_ASM_ACTOR`. `--ui` overrides `ARGUS_UI`.

argus-asm reads environment variables only. It does not read `.env` files. The file `.env.example` in the repository lists placeholder names for reference; copy values into your shell or service manager yourself.

The launcher preference (`gui`, `console`, or `cli`) is stored in `argus-asm/ui.conf` under the per-user configuration directory. Set it with `argus-asm config set-ui gui`, inspect it with `argus-asm config show`, and remove it with `argus-asm config reset-ui`.

## Database location

Resolved in this order:

1. `--db PATH`
2. `ARGUS_ASM_DB`
3. The per-user configuration directory, in a subdirectory named `argus-asm`, with the file `argus-asm.db`:
   - Linux: `$XDG_CONFIG_HOME/argus-asm/argus-asm.db`, or `~/.config/argus-asm/argus-asm.db`
   - macOS: `~/Library/Application Support/argus-asm/argus-asm.db`
   - Windows: `%AppData%\argus-asm\argus-asm.db`

Relative paths are converted to absolute paths. `diagnostics --show-path` includes the path in the report. Without it, the path is hidden.

Run `argus-asm diagnostics` to see which source the path came from. The report says `database config source: environment` (or `flag` or `default`).

## Actor

`--actor` names the user for a command. If `--actor` is absent, `ARGUS_ASM_ACTOR` is used. When no users exist, the actor may be omitted or set to `local`, which acts as an administrator. Once users exist, an actor is required and must name a known user. See [USAGE.md](USAGE.md#roles-and-actors).

## Permissions

- The database file is created with mode `0600` on Unix-like systems.
- A parent directory created by `init` gets mode `0700`.
- On Windows, file modes are not enforced the same way. Keep the database in a folder only your account can read.

## Limits

These limits are fixed in this release. They are listed here so you know what to expect.

| Limit | Value |
|---|---|
| Import file size | 5 MiB (findings CSV or JSON, Nmap XML, PEM file) |
| Findings per import | 10,000 rows |
| Field length in findings | 512 characters |
| Certificates per PEM file | 100 |
| Ports per probe run | 64 distinct |
| Probe timeout | 100ms to 10s (default 2s) |
| Probe concurrency | 1 to 16 (default 4) |
| Lab ports | 1024 to 65535, at most 8 |
| Status reason | 500 characters |
| Search term | at least 2 characters |

## Network

Nothing listens on a network port. The only outbound connections are the TCP connect checks of `probe run --execute`, which run only for allowed assets. The `lab serve` command listens on `127.0.0.1` only, and only while it runs.
