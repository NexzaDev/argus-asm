# Security

This document describes the security properties of argus-asm as implemented. It must agree with the root [SECURITY.md](../SECURITY.md), which explains how to report vulnerabilities. Read [THREAT_MODEL.md](THREAT_MODEL.md) for the reasoning behind each control.

## Purpose and boundaries

argus-asm records assets and findings that you already have, and it tracks their remediation. Its only network activity is the optional TCP connect check, which requires an in-scope asset, an administrator's permission, and an explicit `--execute`. It is for systems you own or are authorized in writing to assess.

## Security properties

| Property | Current behavior |
|---|---|
| Scope before inventory | An asset must fall inside a declared scope. Otherwise the command exits 5. |
| Active checks off by default | Probing needs `asset probe allow` for the asset, a matching scope, and `--execute`. Plans without `--execute` contact nothing. |
| Bounded active checks | At most 64 ports, 16 parallel connections, and timeouts between 100ms and 10s. |
| Listener scope | The lab binds `127.0.0.1` only, uses ports 1024 to 65535, and runs only while the command runs. The main program opens no listener. |
| Import validation | Every row is checked before any write. Imports are all-or-nothing and transactional. |
| Bounded input | Files are limited to 5 MiB. Findings (CSV, JSON, SARIF) are limited to 10,000 rows and 512 characters per field. Nmap files are limited to 10,000 hosts and 10,000 open-port services. PEM files are limited to 100 certificates. |
| Safe XML | Nmap files with an internal DTD subset or entity declarations are refused. |
| Strict JSON | Findings JSON refuses unknown keys and trailing data. |
| Parameterized SQL | All values are bound as parameters. Migrations are embedded static files. |
| Role checks | Viewer, analyst, and administrator roles are checked in the service layer before every operation. Permission failures exit 6. |
| Audit log | Append-only at the database level (triggers). Hash-chained. Readable and verifiable by administrators only. |
| CSV export safety | Cells that could be read as formulas are prefixed with an apostrophe. |
| Restrictive files | On Unix-like systems the database is created with mode `0600`, and a new parent directory with `0700`. |
| Redacted diagnostics | `diagnostics` hides the database path unless `--show-path` is given, and never prints findings or asset values. |
| No telemetry | No data is sent anywhere. |
| Notification delivery | `notify deliver` writes to a local file that you name. It opens no network connection and runs no hook. The file is created with mode `0600`, and a symbolic link is refused. |
| Sealed copies | `db seal` writes an AES-256-GCM copy under a passphrase you supply on standard input. The passphrase is never stored. See [Encrypted copies](#encrypted-copies). |

## Known limitations

- **The actor name is not authenticated.** Roles are a record of who is responsible and a guard against mistakes. They do not protect against someone who can run the program against the database file.
- The audit chain detects casual edits. An attacker with write access to the database can rebuild it.
- The live database is not encrypted. Only the sealed copies made with `db seal` are.
- Domain probes depend on the operating system resolver, so a name can resolve to an address that is outside your scope.
- The build requires Go 1.27.3, which is the current 1.27 patch release. Binaries built with an older Go release can contain standard-library vulnerabilities that the code reaches. See "Vulnerability scanning" below.
- Access control is for a single trusted operator. It is not a security boundary between users. See [THREAT_MODEL.md](THREAT_MODEL.md#access-control-assumptions).

## Handling sensitive files

- Keep the database and exports out of shared folders and version control. `.gitignore` excludes the default demo database path, `*.db` files, `*.sealed` files, and temporary snapshot files, but not every path.
- A notification file (`notify deliver`) lists asset values and finding titles, so it is as sensitive as an export.
- Inputs can contain real hostnames and IP addresses. Use synthetic data in bug reports and tests.
- Remove a demo database when you finish: delete the file.

## Encrypted copies

`db seal` and `db unseal` protect a copy of the database. They do not encrypt the live database, which is plaintext while argus-asm uses it. See [COMMANDS.md](COMMANDS.md#sealed-copies) for the commands.

### Format

A sealed file starts with the 8-byte marker `ARGUSDB1`, then a 4-byte big-endian PBKDF2 iteration count, a 16-byte random salt, and a 12-byte random nonce. The ciphertext follows.

- **Key:** PBKDF2-HMAC-SHA256 of the passphrase and salt, 32 bytes. `db seal` uses 600,000 iterations. When reading, the iteration count must be between 100,000 and 10,000,000.
- **Cipher:** AES-256-GCM. The header (marker, iteration count, salt, and nonce) is authenticated as additional data, so changing any header byte makes decryption fail.
- **Size:** the plaintext is limited to 1 GiB. A sealed file is read whole into memory.
- **Code:** Go standard library only (`crypto/aes`, `crypto/cipher`, `crypto/pbkdf2`, `crypto/rand`). The package is `internal/vault`, with tests for round trips, wrong passphrases, altered headers, and iteration bounds.

### Key management

- **The passphrase is the only key.** argus-asm does not store it, cache it, log it, or derive it from a file. A lost passphrase makes the sealed copy unreadable, and nothing can recover it.
- **Input:** the passphrase is read from the first line of standard input (`--passphrase-stdin`). It is never taken from an argument or an environment variable, and `db` cannot run inside the console. It must be at least 12 bytes. Anything longer is your responsibility: use a long random passphrase from a password manager.
- **Separation:** store the sealed file and the passphrase in different places. Storing them together defeats the purpose.
- **Rotation:** there is no in-place rotation. To change the passphrase, unseal the copy to a new file and seal it again with the new passphrase.
- **Not provided:** key escrow, hardware keys, and operating-system keychain integration.

### Limits

- While argus-asm has the database open, the database is plaintext. `db seal` copies it with SQLite `VACUUM INTO` into a temporary file named `.argus-asm-snapshot-<random>.tmp` in the database's directory. The file is created empty with mode `0600` and removed before the command returns. If the process is killed during sealing, that file can remain. Delete any such file.
- Encryption protects a copy at rest. It does not protect against a process that runs as your user and can read the live database.
- The `db.snapshot` audit entry is written to the live database after the copy is taken, so the sealed copy does not contain it. Run `audit verify` after unsealing to check the audit chain of the restored database.
- `db unseal` checks that the result is an argus-asm database. It does not check that the copy is the most recent one.
- The format has not been reviewed by an outside cryptographer. It uses standard primitives, but the construction is the project's own.

## Reporting a vulnerability

Report vulnerabilities privately through GitHub private vulnerability reporting (the **Report a vulnerability** button on the Security tab), or by email to the backup address in the root [SECURITY.md](../SECURITY.md). Do not open a public issue or post details publicly.

## Vulnerability scanning

Two checks run in CI on every pull request and push to `main`. Both are pinned.

| Check | Version | Toolchain | Command | CI job |
|---|---|---|---|---|
| staticcheck | 2026.2.2 | `go.mod` (Go 1.27.3) | `go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...` | `static-analysis` |
| govulncheck | v1.9.0 | Go 1.27.3 | `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` | `vulnerability-scan` |

staticcheck 2025.1.1 cannot read the export data of Go 1.27 standard libraries, so it was replaced by 2026.2.2, which can.

govulncheck lists only vulnerabilities that the code can reach. The default output also counts findings in packages and modules that the code imports or requires but does not call. `-show verbose` lists those.

### Result with Go 1.27.3

govulncheck v1.9.0 on Go 1.27.3, scanning `./...`: default and `-show verbose` both report **no vulnerabilities**. This covers the standard library and the one required module, `github.com/mattn/go-sqlite3` v1.14.52.

### Reachable standard-library findings and the Go version

The previous build used Go 1.25.13 and was affected by five reachable standard-library vulnerabilities. Go 1.27.3 is the current patch release of Go 1.27, and it clears them and the informational findings listed below.

| Advisory | Package | Reachable from | Fixed in |
|---|---|---|---|
| GO-2026-6088 | `encoding/xml` | `importer.ParseNmapXML` calls `xml.Unmarshal` | 1.25.13, 1.26.6 |
| GO-2026-5972 | `encoding/asn1` | `importer.ParseCertificatesPEM` calls `x509.ParseCertificate`, which calls `asn1.Unmarshal` | 1.25.13, 1.26.6 |
| GO-2026-4971 | `net` (Windows only) | `probe.TCPConnect` calls `net.Dialer.DialContext`; `lab.Serve` calls `net.Listen` | 1.25.10, 1.26.3 |
| GO-2026-4601 | `net/url` | `importer.ParseCertificatesPEM` calls `x509.ParseCertificate`, which calls `url.Parse` | 1.25.8, 1.26.1 |
| GO-2026-4341 | `net/url` | `sqlite.Open` calls `sql.Open`, which calls `url.ParseQuery` | 1.24.12 |

The `go.mod` file declares `go 1.27.3`, so every build and test job uses that patch release. The lowest patched version for the five findings is Go 1.25.13. That line is out of upstream support now, so 1.27.3 was chosen instead. Go 1.27.3 is also the release that the vulnerability scan uses.

Binaries built with older toolchains are affected. Rebuild them with Go 1.27.3.

### Informational findings on Go 1.25.13 (historical)

On Go 1.25.13, the default scan reported 2 findings in packages and 12 in modules that the code does not call. The verbose scan listed 14 more advisories. They were not counted as affected, but they showed that the 1.25 line was out of support. Go 1.26.10 and Go 1.27.3 fix them. The advisories were:

| Advisory | Package | Fixed in |
|---|---|---|
| GO-2026-6809 | `crypto/x509` | 1.26.10, 1.27.3 |
| GO-2026-6604 | `os` (Windows) | 1.26.9, 1.27.2 |
| GO-2026-6599, 6600, 6603, 6605, 6607, 6608, 6609, 6610, 6611, 6612, 6613, 6617 | `stdlib` (net/http, crypto/tls, html/template, and others) | 1.26.9, 1.27.2 |

### Positive control

The scanner was checked against a build with Go 1.24.11, which is older than the fixes above. govulncheck v1.9.0 reported exit 3 and listed the five reachable standard-library advisories, so the scanner and database detect them.

### How the scans were run

- The development sandbox cannot reach `vuln.go.dev`. The scans used a snapshot of `golang/vulndb` at `data/osv` (commit `8f7e390`, 2026-10-27) with `govulncheck -db file:///path/to/data/osv`. CI queries the live database.
- Commands and exit codes are in [TESTING.md](TESTING.md#recorded-results).

The scans do not replace review. A clean result means no known and reachable vulnerability in the scanned toolchain and modules, not that the code is free of bugs.
