# Testing

This page lists the test layers, the exact commands, and what has and has not been run. Exact results of the last recorded run are in the "Recorded results" section. Update that section whenever the checks are run again.

## Prerequisites

- Go 1.27.3 and a C compiler (see [INSTALLATION.md](INSTALLATION.md)).
- Optional: GNU Make.
- Fuzz targets need no extra tools. They run with `go test -fuzz`.

## Commands

| Purpose | `make` target | Plain command |
|---|---|---|
| Format check | `make fmt-check` | `gofmt -l cmd internal migrations` (must print nothing) |
| Format | `make fmt` | `gofmt -w cmd internal migrations` |
| Static checks | `make vet` | `go vet ./...` |
| staticcheck (CI version) | `make staticcheck` | `go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...` |
| Vulnerability scan | `make govulncheck` | `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` |
| Unit and CLI tests | `make test` | `go test -count=1 ./...` |
| Race detector | `make race` | `go test -race -count=1 ./...` |
| Fuzz smoke (10 s per target) | `make fuzz-smoke` | four `go test -run='^$' -fuzz='^Name$' -fuzztime=10s ./internal/importer/` runs (CSV, JSON, Nmap XML, SARIF), see the Makefile |
| Build | `make build` | `go build -trimpath -ldflags "-X github.com/NexzaDev/argus-asm/internal/buildinfo.Version=0.1.0-dev" -o bin/argus-asm ./cmd/argus-asm` |
| Demo walkthrough | `make demo` | builds, then runs the synthetic demo into `build/demo/argus-asm-demo.db` and appends a notification file to `build/demo/outbox.jsonl` |
| Release archive (Unix host) | `make dist` | builds `dist/argus-asm-VERSION-OS-ARCH.tar.gz` with LICENSE, README, and THIRD_PARTY_NOTICES, plus a `.sha256` file. Windows: see the CI job |
| Parser timing (worst case) | none | `go test -run='^$' -bench WorstCase -benchmem ./internal/importer/` |
| Quick check | `make check` | `fmt-check`, `vet`, `test`, `build` |

`make staticcheck` and `make govulncheck` download their tools through the Go module proxy, so they need network access. `make check` does not, and it does not run them. CI runs both (see below).

Run `make check` before opening a pull request. Run `make race` and `make fuzz-smoke` for changes to parsers, the database layer, or concurrent code. Run `make staticcheck` and `make govulncheck` for dependency changes and before a release.

## Continuous integration

`.github/workflows/ci.yml` runs on pull requests and on pushes to `main`. Its jobs are:

| Job | Go version | Checks |
|---|---|---|
| `test (ubuntu-latest)`, `test (macos-latest)`, `test (windows-latest)` | `go.mod` (`go 1.27.3`) | gofmt, vet, race-enabled tests, build, smoke sequence (findings import, SARIF, notification delivery, seal and unseal) |
| `static-analysis` | `go.mod` (`go 1.27.3`) | staticcheck 2026.2.2. The 2025.1.1 release cannot read the export data of Go 1.27 standard libraries. |
| `vulnerability-scan` | `1.27.3` | govulncheck v1.9.0 against the live Go vulnerability database |
| `release-artifact (ubuntu-latest)`, `(macos-latest)`, `(windows-latest)` | `go.mod` (`go 1.27.3`) | builds `-trimpath` binaries with the version `0.1.0-dev+SHORTSHA`, runs `version`, writes an archive (`.tar.gz` on Unix, `.zip` on Windows) with a SHA-256 file, and uploads them as workflow artifacts for 14 days. Nothing is published. |

The former `minimum-go` job was removed. The test matrix runs the exact version in `go.mod`, which is now the same as the version the job used to test.

Line endings: `.gitattributes` forces LF in checkouts. Without it, Windows checkouts with `core.autocrlf` produce CRLF files, and `gofmt -l` reports them.

## Test layers

| Layer | Location | What it covers |
|---|---|---|
| Domain | `internal/domain/domain_test.go` | Hostnames and IP normalization, scope parsing and authorization, severities, dedupe keys, risk factors, lifecycle transitions, RBAC roles |
| Importers | `internal/importer/findings_test.go`, `formats_test.go` | CSV and JSON findings, Nmap XML (including refused DTD and entities), PEM certificates, row limits, byte limits at and one over the maximum, line-numbered errors |
| Fuzz targets | `internal/importer/parsers_fuzz_test.go` | `FuzzParseFindings`, `FuzzParseFindingsJSON`, `FuzzParseNmapXML` |
| Service | `internal/service/service_test.go` | Scope and asset rules, import all-or-nothing behavior, dry runs, roles, first-administrator and last-administrator rules, risk reason requirement, tickets and overdue status, change detection, notifications, probe gating, out-of-scope refusal, plan-only runs, and probe bounds |
| Storage and audit | `internal/repository/sqlite/store_test.go` | Migrations (including repeat runs), missing-database detection, duplicate-key mapping, audit triggers that refuse UPDATE and DELETE, hash chain across reopen, tamper detection |
| CLI | `internal/cli/app_test.go`, `console_test.go` | Command dispatch, flags, exit codes, console behavior (failing lines do not stop the session, `exit` stops it, nested consoles and `-` are refused) |
| Probe | `internal/probe/tcp_test.go` | Detection of a listening loopback port, cancellation |
| Lab | `internal/lab/lab_test.go` | Loopback binding and clean shutdown on context cancel; privileged ports refused |
| Export | `internal/export/export_test.go` | CSV and JSON output, formula neutralization |
| Config | `internal/config/config_test.go` | Database path precedence (flag, environment, default) |

Tests use synthetic data only: `example.test` names and `192.0.2.0/24` addresses. No test contacts a real host. The probe tests connect only to a listener that the test starts on `127.0.0.1`. The lab tests bind to `127.0.0.1` on ports the test chooses.

## Manual checks

These checks are run by hand and are not part of `go test`:

- `lab serve` with a probe in a second terminal (the walkthrough in [USAGE.md](USAGE.md#11-active-checks-optional)).
- The console with a scripted session: `printf 'findings list --limit 2\nexit\n' | argus-asm console --actor root`.
- The migration path from an earlier schema, using `init` on an older database file.

## Recorded results

### Current results: Go 1.27.3 (2026-10-29)

Environment: Linux x86_64 sandbox. Toolchain: go1.27.3 from PyPI `go-bin==1.27.3`, used with `GOTOOLCHAIN=local` so `go.mod` could not pull another toolchain. `GOPROXY=direct` and `GOSUMDB=off` were set because the Go module proxy is not reachable from this sandbox. Tools: staticcheck 2026.2.2 and govulncheck v1.9.0 were built from their GitHub tags. Their `golang.org/x/*` requirements were replaced with the matching `github.com/golang/*` mirrors in a scratch build copy, because the `golang.org` vanity host is not reachable. The vulnerability database is `golang/vulndb` at `8f7e390` (2026-10-27), read from `data/osv` with `-db file://`.

| Command | Exit | Result |
|---|---|---|
| `go mod edit -go=1.27.3`; `go mod tidy` | 0 | `go.sum` unchanged |
| `go build ./...`; `go vet ./...` | 0; 0 | |
| `make check` | 0 | gofmt check, vet, all package tests, build |
| `make race` | 0 | `go test -race -count=1 ./...` |
| `make demo` | 0 | synthetic demo; `audit verify`: 10 chained entries verified |
| `make fuzz-smoke` | 0 | three targets, 10 s each, all PASS |
| CI smoke sequence (`bin/argus-asm`, fresh database) | 0 | `SMOKE_OK`; `audit verify`: 9 chained entries verified |
| `staticcheck` 2026.2.2 `./...` | 0 | no findings |
| `govulncheck` v1.9.0 `-db file://... ./...` | 0 | "No vulnerabilities found." |
| `govulncheck` v1.9.0 `-db file://... -show verbose ./...` | 0 | no advisories listed |
| Control: `govulncheck` v1.9.0 on a copy of the previous commit, `go` directive set to 1.24.11 | 3 | reports GO-2026-4341, GO-2026-4601, GO-2026-4971, GO-2026-5972, GO-2026-6088 |

Not repeated in this run: the lab, probe, console, and schema-upgrade walkthroughs. They ran on Go 1.25.13 (see "Manual checks" below). The code paths they use did not change, but the 1.27.3 runs did not exercise them.

Not run here: any check on macOS or Windows. Those run only in CI, see the CI section.

### Earlier results: Go 1.25.13 (historical)

These runs were made before the change to Go 1.27.3. They are kept as a record. Environment: the same Linux sandbox, with go1.25.13 built from the official tag. `staticcheck` was built from the upstream 2025.1.1 tag.

#### Local runs

| Toolchain | Command | Exit | Notes |
|---|---|---|---|
| go1.25.13 | `make check` | 0 | gofmt, vet, all package tests, build |
| go1.25.13 | `make race` | 0 | `go test -race -count=1 ./...` |
| go1.25.13 | `make demo` | 0 | synthetic demo database |
| go1.25.13 | `make fuzz-smoke` | 0 | three targets, 10 s each, all PASS. The fuzzer's reported rate fell to 0 execs/s near the end of a run. |
| go1.25.13 | `make build`, then the CI smoke sequence | 0 | 9 audit entries verified |
| go1.25.13 | `go test -count=1 -run EnforcesByteLimit -v ./internal/importer/` | 0 | new tests for the Nmap and PEM byte limits |
| go1.25.13 | staticcheck 2025.1.1 (built from the upstream tag; the pinned `golang.org/x/*` versions came from local clones) `./...` | 0 | |
| go1.25.13 | govulncheck v1.9.0, `-db file://` snapshot, `./...` | 0 | "No vulnerabilities found." Summary: affected by 0; 2 in packages and 12 in modules imported but not called |
| go1.25.13 | govulncheck v1.9.0, `-db file://` snapshot, `-show verbose ./...` | 0 | lists the 14 uncalled findings in [SECURITY.md](SECURITY.md#vulnerability-scanning) |
| go1.27.3 | `make check` | 0 | |
| go1.27.3 | `make race` | 0 | |
| go1.27.3 | govulncheck v1.9.0, `-db file://` snapshot, `./...` | 0 | "No vulnerabilities found." |
| go1.27.3 | govulncheck v1.9.0, `-db file://` snapshot, `-show verbose ./...` | 0 | no findings |
| go1.24.11 (control) | govulncheck v1.9.0, `-db file://` snapshot, `./...` | 3 | 5 reachable standard-library findings. Confirms that the scanner detects the issue this change fixes. |

The snapshot is `golang/vulndb` at commit `8f7e390` (2026-10-27), read from `data/osv`. The sandbox cannot reach `vuln.go.dev`. CI uses the live database.

#### Manual checks run against the 1.25.13 binary

- Bootstrap: `init`; `asset list` and `asset list --actor local` both exit 0 before any user exists. `user add local` exits 3 (reserved). `user add root --role viewer` as the first user exits 3. `user add root --role admin` exits 0. After that, `asset list` with no actor exits 6, and `asset list --actor local` exits 6 ("unknown actor"). `asset list --actor root` exits 0.
- Roles: a viewer can list findings (exit 0), and `scope add` and `user add` by the viewer exit 6.
- Search: `search a` exits 3 ("at least 2 characters"). `search TLS` matches a finding title. `search PLATFORM-team` matches an asset by owner. `search admin-ui` matches a service name. Matching is case-insensitive.
- Dedupe: the finding key is asset kind and value, title (trimmed, case-insensitive), and the row's own `source` column (CSV) or field (JSON). The `--source` flag on `findings import` only labels the batch; it does not change row sources. Re-importing the same rows updates them. An uppercase title from the same row source updates the existing finding. The same title with a different row source creates a new finding. Batches are reported per import. Checked against `internal/domain/finding.go` (`DedupeKey`) and `internal/repository/sqlite/findings.go`.
- Nmap: `service import-nmap testdata/demo/nmap.xml` stores the two open ports (22 and 443) and not the closed port (23).
- Lab and probe: `scope add 127.0.0.0/16`, `asset add 127.0.0.1`, `asset probe allow 127.0.0.1`, and `lab serve` in the background. `probe run 127.0.0.1 --ports 18080,18022,19999 --execute --timeout 1s` printed `probed 127.0.0.1: 2 open, 1 closed or filtered` with the two open ports, exit 0. The two services were stored with source `tcp-connect`. A plan for `192.0.2.10` after `asset probe allow` printed the "nothing was contacted" plan line, exit 0. Without the allow, the probe exits 6.
- Console: `printf 'findings list --limit 2\nexit\n' | argus-asm console --actor root` exits 0. `ticket list --overdue` works in the console.
- Upgrade: a database made by the schema 1 binary (commit `7dff44d`, built with go1.25.13) was upgraded by `init` to schema 2. Its scope, asset, and two audit entries were kept, and `audit verify` reported them as legacy. A later import into the upgraded database worked, and `audit verify` then reported the new entries as chained.

#### Runs before the Go 1.25.13 change, not repeated

Before the change to `go 1.25.13`, the module declared `go 1.24.0`. Those runs used a go1.24.13 toolchain built from source (`make check`, `make race`, `make demo`, and `make fuzz-smoke` exited 0, and govulncheck v1.1.4 exited 3 with four findings). They are not repeated here, because the sandbox was reset and the toolchain was not kept. Their results are the basis for the decision in [SECURITY.md](SECURITY.md#vulnerability-scanning).

### CI runs

Run 40821785897 on commit `53f9576` (the Go 1.27.3 change). This run was on the earlier branch `arena/5f6a4bc7-argus-asm`. Its commit is not in the history of `arena/2578041a-argus-asm`, so these results do not cover this branch:

| Job | Result |
|---|---|
| `test (ubuntu-latest, Go from go.mod)` | success |
| `test (macos-latest, Go from go.mod)` | success |
| `test (windows-latest, Go from go.mod)` | success |
| `staticcheck (Go from go.mod)` | success |
| `govulncheck v1.9.0 (Go 1.27.3)` | success |

The `test` and `staticcheck` jobs take `go.mod`, which is `go 1.27.3`. The patch version that `setup-go` resolved was not read from the job logs. Those logs could not be downloaded from the sandbox. Each job's "Record Go toolchain" step logs the version.

Earlier runs, on the Go 1.25.13 line: run 40759228557 on `bf929e0` passed all six jobs, including the separate `minimum-go` job, which has since been removed. Run 40752506213 on `abb4df1` and run 40754308968 on `c0d9ab2` passed with `go 1.24.0`. Runs 40760415532 (on `9e343ea`) and 40763737472 (on `a2f38e3`) cancelled the macOS job before a runner was assigned, so macOS was not re-tested on those heads. Re-running that job through the API returned HTTP 403 with this token.

## Round 3 results: Go 1.27.3 (2026-10-29)

Environment: Linux x86_64 sandbox, 2 vCPU Intel Xeon Processor at 2.60 GHz, kernel `Linux 6.1.158+`. Toolchain: go1.27.3 from PyPI `go-bin==1.27.3`, `GOTOOLCHAIN=local`, `GOPROXY=direct`, `GOSUMDB=off`. Tools: staticcheck 2026.2.2 and govulncheck v1.9.0 binaries built from their upstream tags in earlier rounds. The vulnerability database is the `golang/vulndb` snapshot at `8f7e390` (2026-10-27), read with `-db file://`. The sandbox cannot reach `honnef.co` or `vuln.go.dev`, so `make staticcheck` and `make govulncheck` were not run through `make`.

| Command | Exit | Result |
|---|---|---|
| `gofmt -l cmd internal migrations .` | 0 | no files listed |
| `go vet ./...` | 0 | |
| `go test -count=1 ./...` | 0 | all packages |
| `make check` | 0 | format check, vet, tests, build |
| `make race` | 0 | `go test -race -count=1 ./...` |
| `make demo` | 0 | synthetic demo, including the SARIF import, notification delivery to `build/demo/outbox.jsonl`, and `audit verify`: 12 chained entries |
| `make fuzz-smoke` | 0 | four targets, 10 s each, all PASS. The fuzzer's reported rate fell to 0 execs/s near the end of each run, as in the earlier Go 1.25.13 run. |
| `staticcheck-2026.2.2 ./...` | 0 | no findings |
| `govulncheck-v1.9.0 -db file:///tmp/tools/vulndb/data/osv ./...` | 0 | "No vulnerabilities found." |
| `govulncheck-v1.9.0 -db file:///tmp/tools/vulndb/data/osv -show verbose ./...` | 0 | "No vulnerabilities found." |
| `make dist` | 0 | `argus-asm-0.1.0-dev-linux-amd64.tar.gz` and its `.sha256` checked with `sha256sum -c`: OK |
| CI smoke script, extracted from `ci.yml`, fresh database | 0 | includes SARIF import, notification delivery (second run delivers 0), seal and unseal round trip. The restored copy reports 12 chained entries and the live database 13, because the `db.snapshot` entry is written after the copy is taken. |
| CI Unix packaging script, extracted from `ci.yml`, with a sample SHA | 0 | archive and checksum written; `sha256sum -c`: OK |

Final re-run after the last source change (comments removed from Go sources, which changed no behavior): `gofmt -l` (no output), `go vet ./...`, `go test -count=1 ./...`, `make check`, `make race`, `make demo`, `make fuzz-smoke`, `staticcheck-2026.2.2 ./...`, `govulncheck-v1.9.0` (default and `-show verbose`), `make dist` with `sha256sum -c`, and the CI smoke script all exited 0. The worst-case benchmark was rerun; the table above shows that run. The logs are in the sandbox at `/tmp/verify2/r3final2/`, which is not part of the repository.

Unit and CLI tests added in this round: SARIF importer and fuzz target, notification delivery (service and CLI), the outbox file checks, the migrations from schema 2 and the SARIF migration, the vault (`internal/vault`), `db seal` and `db unseal` (round trip, wrong passphrase, refused roles, short passphrase, console refusal), the Nmap open-service cap, and the worst-case inputs.

### Parser timing, worst-case inputs

`internal/importer/worstcase_bench_test.go` builds the largest accepted input for each format: 10,000 rows of 440-character text for CSV, JSON, and SARIF, and 10,000 hosts with one open port each for Nmap. `TestWorstCaseInputsAreAccepted` checks that each input is under the 5 MiB cap and is accepted with the full count, so the benchmarks measure accepted input only.

Command: `go test -run='^$' -bench WorstCase -benchmem -benchtime=2s ./internal/importer/`. Output, one run:

| Benchmark | ns/op | MB/s | B/op | allocs/op |
|---|---|---|---|---|
| `ParseFindingsCSVWorstCase` | 5,770,548 | 819.68 | 8,838,991 | 20,037 |
| `ParseFindingsJSONWorstCase` | 57,582,053 | 90.13 | 50,722,128 | 140,076 |
| `ParseFindingsSARIFWorstCase` | 30,941,680 | 157.40 | 40,159,043 | 50,159 |
| `ParseNmapXMLWorstCase` | 86,557,421 | 21.90 | 40,280,274 | 880,094 |

These are the final figures, from the run after comments were removed from the sources. Two runs of the same command differed by up to about 25% (SARIF), so treat them as order-of-magnitude numbers from a shared sandbox, not a guarantee for other hardware. The first run of the Nmap benchmark used two open ports per host, which is 20,000 services. That took about 130 ms and 1,270,098 allocations. Such a file now exceeds the 10,000-service cap, so the benchmark uses one port per host.

### Walkthroughs

The commands in [USAGE.md](USAGE.md) were run in order on a fresh database, with the outputs compared to the text blocks. The lab and probe steps used a background `lab serve` on loopback. Every documented output matched except these, which were corrected in the docs:

- Section 2 did not add `api.example.test`, so the section 5 imports failed with "not in the inventory". The step now adds it.
- Section 9 said "batch 1 and batch 2". The batch numbers count every import, so the second CSV is batch 4 after the JSON and SARIF imports. The text now says so.
- Section 9 used a file that was not in the repository. `testdata/demo/findings-second.csv` was added, and it reproduces the documented output.
- The notification text in section 9 said notifications were never delivered outside the database. That is no longer true.
- The roles table did not mention notification delivery or sealed copies.

Commands that were not run by the walkthrough: the sealed-copy example with `read -rs`, which needs an interactive shell. It was tested through the CI smoke script, which uses the same flags with a generated passphrase. The PowerShell form is not documented.

### Not verified in Round 3

- The Windows packaging step in the `release-artifacts` job. PowerShell is not installed in the sandbox, and the job has not run in CI for this commit.
- Test, static-analysis, and vulnerability jobs for this commit. They run only when the branch is pushed.
- The exact patch version `setup-go` resolved on each runner. CI logs could not be downloaded from the sandbox, so the version is recorded only as `go.mod` (1.27.3) and by each job's "Record Go toolchain" step.
- macOS and Windows behavior of the new code. Those platforms were not run locally.

## What is not verified

- Architectures other than amd64 on Linux and Windows, and arm64 on any platform. No runner or local machine was used for them.
- Local runs on Windows or macOS. Their results come only from GitHub-hosted runners in CI.
- The exact patch versions that CI used for the `go.mod` jobs. Each job logs them in its "Record Go toolchain" step. The sandbox could not download the job logs, so they are not recorded here.
- The live vulnerability database was not reached from the sandbox. Local govulncheck runs used a snapshot. CI queries the live database.
- The staticcheck 2026.2.2 and govulncheck v1.9.0 binaries used locally were built from source with replaced `golang.org/x/*` dependencies. CI runs them with `go run ...@version` through the Go module proxy, which the sandbox cannot reach. CI results for these versions are the authoritative check.
- The lab, probe, console, and schema-upgrade walkthroughs were not run on Go 1.27.3.
- No worst-case parse timing has been measured. The byte-limit tests check size rules, not time. No benchmark exists. The fuzz smoke test shows the fuzzer's reported execution rate dropping to 0 near the end of each 10-second run. The targets still passed, and the cause was not investigated.
- No performance or load measurements exist.
- The Go 1.27.3 toolchain has not been built and tested on a non-Linux machine locally. The CI matrix covers Windows and macOS on `go.mod`'s toolchain only.

## Round 4 results: launcher and GUI feasibility (Go 1.27.3, 2026-10-29)

Environment: Linux x86_64 sandbox. Toolchain: go1.27.3 linux/amd64 from PyPI `go-bin==1.27.3`, `GOTOOLCHAIN=local`, `GOPROXY=direct`, `GOSUMDB=off`. Only Linux was run this round. Note: CI run 40866937827 (`8808e1f`) failed only the staticcheck job. Its log could not be downloaded from GitHub (EOF), so the finding was reproduced locally with the staticcheck 2026.2.2 binary built from `dominikh/go-tools` tag `2026.2.2` with replaced `golang.org/x/*` dependencies. macOS and Windows were not run locally; CI is the check for them.

Commands run from the repository root, with exit codes:

| Command | Exit | Result |
|---|---|---|
| `make fmt-check` | 0 | gofmt clean |
| `make vet` | 0 | `go vet ./...` clean |
| `make test` | 0 | all packages pass, including new `internal/config` UI tests and `internal/cli` launcher and config tests |
| `make race` | 0 | race detector pass, same packages |
| `make build` | 0 | binary built with `-trimpath` |
| staticcheck 2026.2.2 (built from the upstream tag in the sandbox; see note) | 1, then 0 | First run reported SA4006 in `internal/cli/launcher_test.go`, the same finding CI reported for `8808e1f`. The test was fixed; the second run is clean |

Manual run of the built binary with `XDG_CONFIG_HOME` pointed at a temporary directory:

- `config show` printed `ui: cli (source: default)`, exit 0.
- `config set-ui console`, then `config show` printed `ui: console (source: stored)`, exit 0.
- Bare `argus-asm` with `ui=console` started the console; `exit` on stdin ended it, exit 0.
- `config reset-ui` removed the preference (exit 0); a second run printed `nothing to reset` (exit 0).
- `--ui console init --db FILE` exited 2 (`--ui` cannot be combined with a command).
- Bare `argus-asm` with no preference printed usage and exited 2.

GUI feasibility probe (outside the repository, in a scratch module): Fyne v2.8.2 was extracted from its GitHub tarball and compiled with `go build -tags ci` through directory replacements for its vanity-host dependencies (`fyne.io/systray`, several `golang.org/x/*`, `gopkg.in/*`). The build exited 0. This shows the code compiles here but does not make a desktop build, and it produced no `go.sum` for the repository. The desktop build needs cgo with OpenGL and the platform toolkit. Those are not available in this sandbox, so the GUI was not added. See [ROADMAP.md](ROADMAP.md) for the decision record.

## Round 5 local checks (milestones 1 and 2, sandbox)

Sandbox: Linux x86_64, Go 1.27.3, no GL headers, no display, no proxy.golang.org. Module resolution used a sandbox-only workspace file that points `fyne.io/fyne/v2 v2.8.2` at a local copy, so these results do not prove the repository's `go.sum` is complete.

- `gofmt -l .`: empty.
- `make check` (fmt-check, vet with `ci`, test with `ci`, build-nogui): exit 0.
- `go test -count=1 -tags nogui ./...`: exit 0.
- `go test -count=1 -tags ci ./...`: exit 0. Includes `internal/gui` headless tests: launcher chooser, severity contrast, icon resolution, external-link allowlist, docs copies, table filter and sort, every category builds.
- `ARGUS_SCREENSHOT_DIR=... go test -tags ci -run TestWriteDesignScreenshots ./internal/gui/`: exit 0, 12 PNGs written with demo data. Viewed the Overview image; the other images are not reviewed.
- `go list -tags nogui -deps ./cmd/argus-asm | grep fyne.io/fyne`: no output.

Not verified in the sandbox: the default GUI build, a real window on any OS, native notifications, macOS and Windows. These run only in GitHub Actions (`ci.yml`: test, nogui-build, gui-build, design-screenshots, static-analysis, vulnerability-scan, release-artifacts).

## Round 6 local checks (M4 and M5, sandbox, Go 1.27.3)

Environment: Linux sandbox. The module proxy `proxy.golang.org` returned EOF for `fyne.io/fyne/v2` and for `fyne.io/systray`, and `fyne.io/*` vanity hosts are unreachable. So the full Fyne module graph cannot be downloaded here.

### Clean dependency tree (no `go.work`, `GOWORK` unset)

| Command | Exit | Result |
|---|---|---|
| `go build -tags nogui -o /tmp/argus-nogui ./cmd/argus-asm` | 0 | builds |
| `gofmt -l cmd internal migrations` | 0 | no output |
| `go vet -tags nogui ./...` | 0 | clean |
| `go test -count=1 -tags nogui ./...` | 0 | all packages ok |
| `go mod verify` | 1 | cannot download `fyne.io/fyne/v2@v2.8.2` (proxy EOF) |
| `go list -m all` | 1 | cannot download `fyne.io/fyne/v2@v2.8.2` (proxy EOF) |

`go.sum` contains an h1 hash for both the module and its `go.mod` for all 36 modules in `go.mod`. This was checked by script, not by `go mod verify`. It does not prove that `go.sum` matches a tidy graph. Superseded in Round 7 and in the GitHub run described in the "CI on this branch" section.

### With the sandbox workspace (`GOWORK=/tmp/ws/go.work`)

These use directory replacements in `/tmp`. They show that the GUI code compiles and its tests pass. They do not prove the repository's dependency setup.

| Command | Exit |
|---|---|
| `go vet -tags ci ./...` | 0 |
| `go test -count=1 -tags ci ./...` | 0 |
| `staticcheck -tags ci ./...` (honnef.co/go/tools 2026.2.2 built in /tmp) | 0 |
| `staticcheck -tags nogui ./...` | 0 |
| `go test -race -count=1 -tags nogui ./...` | 0 |
| `go test -race -count=1 -tags ci ./...` | 0 |
| `go test -count=1 -tags ci -run TestWriteDesignScreenshots ./internal/gui/` with `ARGUS_SCREENSHOT_DIR=/tmp/shots8` | 0 (12 PNGs) |

Not run: `govulncheck`. `golang.org/x/vuln` is fetched through the module proxy, and the vulnerability database `vuln.go.dev` is unreachable from the sandbox. `staticcheck` default tags (needs Wayland and GL headers) is not run.

### Notifier smoke tests (`nogui` binary)

- `notifier status` exit 0.
- `notifier install` without `--yes` exit 2, prints the consent text.
- `notifier run --interval 1s` exit 2 (below the 1-minute minimum).
- With `HOME` and `XDG_CONFIG_HOME` pointed at a temporary directory: `notifier install --yes --detail detailed` exit 0, wrote a 0600 desktop entry with a quoted `Exec` line. `notifier status` reports it installed. `notifier uninstall` exit 0, and a second uninstall exit 0 with "not installed".
- Not run: `notifier run` against a real desktop notification service. `notify-send` is not installed in the sandbox, so OS display is unverified.

### Icon assets (checked by `TestIconAssetsAreWellFormed`)

ICO sizes 16, 24, 32, 48, 64, 128, 256. ICNS has chunks `ic07` to `ic10`. PNG sizes 16 to 1024. Linux desktop entry `Icon=argus-asm` with `assets/icon/linux/argus-asm.png`. Only the Linux desktop file was checked. Taskbar and dock appearance is not verified.

### Workflow files

- `.github/workflows/release.yml` and `.github/workflows/go-sum.yml` parse as YAML (PyYAML 6.0.3). Superseded in Round 7: actionlint was built from source and run with ShellCheck, see below.

### CI on this branch (checked from run metadata, not logs)

Run metadata was read with `gh run view --json jobs`. Logs were not downloaded.

- Run 40914349608 (`go.sum refresh`, `push`, commit `944a04f`): success. Its tidy step ran `go mod tidy` and `go mod verify` on a GitHub runner, using Go's default proxy and checksum settings. The bot then committed `go.mod` and `go.sum` (commit `a64dc88`, which pins `golang.org/x/image` v0.47.0).
- Run 40918812174 (`CI`, commit `0055668b`). These jobs passed: `test (ubuntu-latest)`, `test (windows-latest)`, `GUI build (ubuntu-latest)`, `GUI build (windows-latest)`, `nogui build and CLI smoke` (Ubuntu and Windows), `design screenshots`, `govulncheck v1.9.0 (Go 1.27.3)`, `release artifact (ubuntu-latest)`, and `release artifact (windows-latest)`. The Windows release-artifact job ran `Build and package (Windows)`. `staticcheck` failed on findings. Later commits address them (for example `1b1668e`). staticcheck passes locally on HEAD, but it has not run on GitHub since then. The macOS jobs started no steps.
- Run 40914362311 (commit `944a04f`, cancelled). Passed before the cancellation: `test (ubuntu-latest)`, `GUI build (ubuntu-latest)`, `release artifact (ubuntu-latest)`, `nogui build and CLI smoke` (Ubuntu and Windows), and `design screenshots`.
- Runs `e39b86c`, `19ae2d8`, `21d6263`, `5ab79ad` and the run for `a0d7745` (40929691482, still queued) started no steps. The macOS jobs in every run started no steps.
- `release.yml`: never run. There is no workflow run for it.
- `go-sum.yml`: still enabled on GitHub. Disabling it returned HTTP 403 to the sandbox token. It is `workflow_dispatch` only in the repository.

## Round 7 local checks (filter label, licenses, go.sum, release workflow)

Environment: Linux sandbox, Go 1.27.3. Commands that build the GUI use the sandbox-only workspace `GOWORK=/tmp/ws/go.work`, which replaces `fyne.io/fyne/v2` with a local copy of v2.8.2. Results under that workspace show that the code compiles and the tests pass. They do not prove the repository's dependency setup.

### Gates

| Command | Exit | Result |
|---|---|---|
| `gofmt -l .` | 0 | no output |
| `go vet -tags ci ./...` | 0 | clean (`GOWORK=/tmp/ws/go.work`) |
| `go vet -tags nogui ./...` | 0 | clean (`GOWORK=/tmp/ws/go.work`) |
| `go build -trimpath -tags nogui -o /tmp/argus-nogui ./cmd/argus-asm` | 0 | builds |
| `go test -count=1 -tags ci ./...` | 0 | 13 packages ok |
| `go test -count=1 -tags nogui ./...` | 0 | all packages ok |
| `go test -race -count=1 -tags nogui ./...` | 0 | all packages ok |
| `go test -count=1 -tags ci -run 'TestDocsCopiesMatchRepository\|TestNotificationFilterLabelsFitTheirSelects\|TestIconAssets' -v ./internal/gui/` | 0 | all three PASS |
| `staticcheck -tags ci ./...` (2026.2.2, built in /tmp) | 0 | no findings |
| `staticcheck -tags nogui ./...` | 0 | no findings |

Not run: `staticcheck` with the default tags (needs GL and Wayland headers, which this sandbox does not have). `govulncheck`: `vuln.go.dev` is unreachable, so it is BLOCKED here.

### Filter label regression

`TestNotificationFilterLabelsFitTheirSelects` checks that the Notifications filter select is at least as wide as its longest label. A bare `Select` measured about 128 px and the longest label needed about 149 px; the `fitSelect` wrapper measured about 153 px. The test was run with the fix (pass) and with the wrapper removed (fail). A design screenshot of the Notifications view shows the full "All notifications" label.

### go.sum

The sandbox cannot reach `proxy.golang.org`, `sum.golang.org`, or `vuln.go.dev`, so the check below does not use them.

1. `go.sum` has 81 lines, for 41 module versions: 36 modules required by `go.mod`, plus 5 modules that only appear in the module graph (`github.com/go-text/typesetting-utils`, `github.com/kr/text`, `github.com/niemeyer/pretty`, and two versions of `gopkg.in/check.v1`).
2. Each module was checked against its GitHub source archive, with the same hash function Go uses (`h1:` over the zip or `go.mod` file list). Results: the zip hash matched for 35 of 36 required modules. `github.com/go-gl/glfw/v3.4/glfw` has no tag archive on GitHub (HTTP 404), so its zip hash was checked against the sandbox module cache's `.ziphash`, and it matched. The `go.mod` hash matched for 32 of the 36 required modules. The other four (`go-spew`, `gobmp`, `nfnt/resize`, `go-difflib`) did not reproduce from the GitHub archive. The sandbox module cache holds the `.mod` file the go command downloaded for each of them, and its hash matches `go.sum`. The five graph-only modules were checked the same way; `gopkg.in/check.v1` has no `go.mod` in its repository, so the expected file is the synthesized `module gopkg.in/check.v1` line, and both of its `go.mod` hashes match `go.sum`.
3. A local GOPROXY (127.0.0.1, serving the module content from the GitHub archives above, with no direct fallback) was used with an isolated copy of the module and a separate module cache: `GOTOOLCHAIN=local GOWORK=off GOSUMDB=off GOPROXY=http://127.0.0.1:8765 GOMODCACHE=<tmp> GOFLAGS=-mod=mod go mod tidy` → exit 0. `go.sum` and `go.mod` were byte-identical afterwards. Go's checksum check was on: an earlier wrong zip produced `SECURITY ERROR ... checksum mismatch`.
4. `go mod verify` in the same copy → `all modules verified`, exit 0.

This is strong evidence, but it is not the real proxy check. Still open: `GOTOOLCHAIN=local go mod tidy` and `git diff --exit-code go.mod go.sum` on a machine that can reach `proxy.golang.org` and `sum.golang.org` (see RELEASING.md).

The proxy and the scratch copy are in /tmp and are not committed.

### Licenses

- `THIRD_PARTY_NOTICES.md`: 34 transitive modules, with the license read from each module's LICENSE file.
- `go list -deps ./cmd/argus-asm` in the sandbox: linux/amd64 default build has 29 module lines (including the main module), and `fyne.io/systray` is in it. For windows/amd64 and darwin/amd64 (`CGO_ENABLED=1`), the default build has the same module set except `github.com/rymdport/portal`, which is Linux-only. `-tags nogui` has only `mattn/go-sqlite3` from this list.
- `scripts/collect-licenses.sh /tmp/licout ./cmd/argus-asm` → exit 0, 28 modules with license files. Run on `./internal/buildinfo` (no third-party modules) → exit 1, as designed.

### Release workflow

- `release.yml` and `ci.yml` and `go-sum.yml` were checked with `actionlint` v1.7.12 built from its GitHub source, with `shellcheck` 0.11.0 (PyPI `shellcheck-py`) on the path: exit 0 after one fix. The release asset of actionlint could not be downloaded, so the binary is from source, not from the release.
- The Linux step that writes the archive listing was run against a fake `fyne-out` in `/tmp` (exit 0). Under `pipefail`, `tar | head` returned 141, so the listing is now written to a file first.
- The release-file count was checked: `find ... -name 'argus-asm-*' | wc -l` is 3 with three archives and 4 with four.
- Not run: the Windows step (no PowerShell here), the full `release.yml` on GitHub, and `fyne package` on any OS.

## Round 8 final readiness review (commit `a0d7745` plus documentation fixes)

Environment: Linux sandbox, Go 1.27.3, `GOWORK=/tmp/ws/go.work` for the Go commands. Exit codes are from the shell.

| Command | Exit | Result |
|---|---|---|
| `gofmt -l .` | 0 | no output |
| `go vet -tags ci ./...` | 0 | clean |
| `go vet -tags nogui ./...` | 0 | clean |
| `go test -race -count=1 -tags ci ./...` | 0 | 13 packages ok |
| `go test -race -count=1 -tags nogui ./...` | 0 | 13 packages ok |
| `go test -count=1 -tags ci ./...` | 0 | 13 packages ok |
| `go test -count=1 -tags nogui ./...` | 0 | all packages ok |
| `go test -count=1 -tags ci -run TestDocsCopiesMatchRepository -v ./internal/gui/` | 0 | PASS |
| `/tmp/staticcheck -tags ci ./...` (2026.2.2) | 0 | no findings |
| `/tmp/staticcheck -tags nogui ./...` | 0 | no findings |
| `go build -trimpath -tags nogui ./cmd/argus-asm` | 0 | builds |
| `actionlint -shellcheck=$(which shellcheck) .github/workflows/*.yml` (actionlint v1.7.12 built from source; ShellCheck 0.11.0) | 0 | no findings |
| `shellcheck -s bash scripts/collect-licenses.sh` | 0 | no findings |
| `bash scripts/collect-licenses.sh <tmp> ./cmd/argus-asm` | 0 | 28 modules |
| `bash scripts/collect-licenses.sh <tmp> ./internal/buildinfo` | 1 | expected: no third-party modules |
| `gh workflow disable go-sum.yml` | 1 | HTTP 403, Resource not accessible by integration |
| `curl https://vuln.go.dev` | 35 | unreachable, so govulncheck cannot run locally |

Not run in this round: `govulncheck` locally (blocked, above), `staticcheck` with the default tags (needs GL and Wayland headers), `go mod tidy` or `go mod verify` against the real proxy from the sandbox (blocked), Windows and macOS builds, `release.yml` on GitHub, and any CI job on the current HEAD (its run has started no jobs).

Findings from the review: the docs contradicted the CI record and the go.sum history. They are corrected in this round. Code and workflow files had no confirmed defects. The Windows notifier's `-ExecutionPolicy Bypass` flag is now documented in COMMANDS.md. Residual risk, not changed: the release draft job uses an existing tag if one exists, so the manual tag check in RELEASING.md is still needed.
