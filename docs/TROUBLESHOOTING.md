# Troubleshooting

Each heading is a symptom you might see. Exit codes are listed in [COMMANDS.md](COMMANDS.md#exit-codes).

## Exit code 4: database is not initialized

```text
error: database is not initialized at /tmp/none/x.db
hint: run 'argus-asm init' to create the local database
```

Run `argus-asm init` with the same `--db` or `ARGUS_ASM_DB` value. Use `argus-asm diagnostics` to see which path and source are being used.

## Exit code 2: usage error

```text
error: flag provided but not defined: -bogus
run 'argus-asm help' for usage
```

Each command prints its own flags when it fails, and `argus-asm <command> -h` prints them on request. Flags go after the command arguments, for example `findings list --limit 3`.

## Exit code 3: invalid input

### Domain or IP rejected

```text
error: domain "bad_name!" must contain at least two labels
```

Use a fully qualified name such as `www.example.test`. Hostnames may contain letters, digits, and hyphens. Underscores and punctuation are refused. IP addresses must be valid IPv4 or IPv6 addresses.

### CIDR "too broad"

```text
error: CIDR "10.0.0.0/8" is too broad (minimum prefix length is /16)
```

Authorize a narrower range, or add several narrower ranges. The limit exists so that one command cannot authorize a very large part of the address space by accident.

### "already exists"

```text
error: already exists: scope example.test
```

The record is already there. Use `scope list` or `asset list` to confirm. Nothing was changed.

### Import: "row(s) rejected; nothing was imported"

```text
line 2: unknown severity "nope" (expected critical, high, medium, low, or info)
error: validation failed: 1 row(s) rejected; nothing was imported
```

Each rejected row is listed with its line number. Fix every listed line and run the import again. Use `--dry-run` first to check a file without writing anything. A row whose asset is not in the inventory is also rejected, with `asset ... is not in the inventory; add it first`. Add the asset (inside a scope) and import again.

### Unknown filter values

`findings list --severity nope` exits 3 with `unknown severity "nope"`. Valid severities are `critical`, `high`, `medium`, `low`, and `info`.

## Exit code 5: outside authorized scope

```text
error: outside every authorized scope: ip 203.0.113.99 is not inside any registered scope; add a scope first
```

This is the scope rule working. Add a scope only if you are authorized to assess that range, in writing. Do not add a scope to make a command succeed for a host you do not control.

## Exit code 6: permission denied

```text
error: permission denied: vera (role viewer) lacks administer permission
```

Use an actor with the needed role, or ask an administrator to change the role with `user role`. Roles are listed in [USAGE.md](USAGE.md#roles-and-actors).

```text
error: permission denied: an actor is required; pass --actor or set ARGUS_ASM_ACTOR
```

Once users exist, name the acting user with `--actor NAME` or `ARGUS_ASM_ACTOR`.

## Probe refused

`probe run --execute` refuses to contact an asset until an administrator runs `asset probe allow <asset>`. Without `--execute` the command only prints a plan and contacts nothing. The scope is checked again at run time, so an asset that no longer falls inside a scope exits 5 before any connection is attempted.

## Lab will not start

- `lab serve --ports 80` exits 3: ports must be between 1024 and 65535.
- `lab serve` with a port already in use exits 1 with `bind: address already in use`. Stop the other process or choose other ports.
- The lab listens on `127.0.0.1` only. Connect to `127.0.0.1`, not to another address.

## Standard input does not work

`findings import` and `service import-nmap` accept `-` to read standard input. The console refuses `-`, because its own commands come from standard input. Inside the console, pass a file path instead.

## Exit code 1: unreadable database

Exit code 1 is a general failure, not covered by the codes above. Database errors are the most likely cause. Check that the file exists, that your account can read and write it, and that it is not on a network share. Keep a backup before you try repairs (see below).

## Database is locked

SQLite reports `database is locked` when another process holds the database for longer than 5 seconds. Close the other `argus-asm` process or console session, then retry. Do not share one database file between machines.

## Audit chain does not verify

`audit verify` reports whether the hash chain is intact. Entries written before schema 2 have no hash and are reported as legacy entries, which is expected after an upgrade. A broken chain means that an audit row was changed outside the tool, for example by someone who dropped the triggers. Keep the database, do not run further writes, and investigate.

## Build fails: cgo or C compiler not found

The SQLite driver needs cgo. Install a C compiler:

- Linux: `gcc` from your package manager.
- macOS: `xcode-select --install`.
- Windows: MinGW-w64 GCC on `PATH` (for example from MSYS2).

Then run the build again. See [INSTALLATION.md](INSTALLATION.md#prerequisites).

## Windows: path or quoting problems

Use `.\bin\argus-asm.exe` in PowerShell. Quote paths that contain spaces. Standard input may be piped in PowerShell with `Get-Content file.csv | .\bin\argus-asm.exe findings import --source demo -`. This pipe has not been verified on Windows. Windows does not apply Unix file modes, so keep the database in a folder only your account can read. See [PLATFORM_SUPPORT.md](PLATFORM_SUPPORT.md).

## Back up and reset

Copy the database while no argus-asm process is running. If a `-wal` file exists next to the database, copy it too, or close all processes first so SQLite can checkpoint.

```sh
cp "$ARGUS_ASM_DB" "$ARGUS_ASM_DB.bak"
```

There is no reset command. To reset, stop all processes, delete the database file and any `-wal` or `-shm` files beside it, then run `argus-asm init`. Keep the backup until you have confirmed the new database is what you want.

## Collecting information for a bug report

1. Run `argus-asm diagnostics` and `argus-asm about`. The diagnostics output hides the database path unless you pass `--show-path`.
2. Reproduce the problem with the synthetic fixtures in `testdata/demo/` if you can.
3. Include the exact command, the exit code, and the standard-error text. Redact real hostnames, IP addresses, and client names before sharing.
