# Installation

argus-asm is built from source. No binary release has been published yet.

## Prerequisites

| Requirement | Why | Notes |
|---|---|---|
| Go 1.27.3 | Build toolchain (`go.mod` declares `go 1.27.3`) | Free from https://go.dev/dl/. Older Go releases have reachable standard-library vulnerabilities; see [SECURITY.md](SECURITY.md#vulnerability-scanning). |
| A C compiler (cgo) | The SQLite driver `github.com/mattn/go-sqlite3` is built with cgo | Linux: `gcc` from your package manager. macOS: Xcode Command Line Tools (`xcode-select --install`). Windows: a MinGW-w64 GCC (for example from MSYS2) on `PATH`. |
| Git | To clone the repository | Optional if you download a source archive. |
| GNU Make | Convenience targets in the `Makefile` | Optional. Every target has a plain `go` equivalent below. |

Nothing else is required. The tool does not need accounts, API keys, a database server, or network access at runtime. The only command that opens network connections is `probe run --execute`, and it contacts only assets that are in scope and allowed for probing.

## Clean install from source

These steps are the same on all platforms, except where noted.

```sh
git clone https://github.com/NexzaDev/argus-asm.git
cd argus-asm
go build -trimpath -o bin/argus-asm ./cmd/argus-asm
./bin/argus-asm version
```

On Windows PowerShell, use `go build -trimpath -o bin\argus-asm.exe .\cmd\argus-asm` and run `.\bin\argus-asm.exe version`.

With `make` installed, `make build` runs the same build with the version string set from `VERSION` (default `0.1.0-dev`).

## Verify the install

```sh
./bin/argus-asm help
./bin/argus-asm about
./bin/argus-asm diagnostics
```

`diagnostics` reports `database status: not initialized` until you run `init`. That is expected on a fresh install.

## Optional: put the binary on your PATH

Copy `bin/argus-asm` (or `bin\argus-asm.exe`) to a directory on your `PATH`, such as `~/.local/bin` on Linux or `/usr/local/bin` on macOS. The tool keeps its data in the per-user configuration directory, so it does not need to be installed in a system location.

## First run

```sh
./bin/argus-asm init
```

This creates the database at the default location described in [CONFIGURATION.md](CONFIGURATION.md) and applies the schema. It is safe to run again.

## Upgrade

Pull the latest source, rebuild, and run `init` again. `init` applies any new migrations and leaves existing data in place. Back up the database file before upgrading (see [TROUBLESHOOTING.md](TROUBLESHOOTING.md#back-up-and-reset)).

## Uninstall and reset

Delete the binary you built. To remove all local data, delete the database file and its `-wal` and `-shm` companions (see [TROUBLESHOOTING.md](TROUBLESHOOTING.md#back-up-and-reset)). The tool creates nothing else outside the database path.

## Platform notes

See [PLATFORM_SUPPORT.md](PLATFORM_SUPPORT.md) for which operating systems and architectures have been verified, and which have only CI configuration.
