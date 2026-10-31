# Threat model

This model covers `0.1.0-dev`. Update it in the same change as any feature that adds data flows, network behavior, or access rules.

## What the tool is for

argus-asm records the assets, services, certificates, and findings of systems you own or are authorized in writing to assess, and it tracks their remediation. It is not a scanner of arbitrary networks. Active checks are limited to assets that are inside a declared scope and that an administrator has explicitly allowed.

## Assets to protect

- The inventory itself: hostnames, IP addresses, services, and certificates of your environment. This data is sensitive for attackers doing reconnaissance.
- Finding details and their risk decisions, including accepted risks and their reasons.
- Ticket owners and ownership records.
- The audit log.
- The database file, which contains all of the above.

## Trust boundaries

1. **Operating-system user to database file.** Anyone who can read or write the file can read or change everything in it. Protection is the file permission (`0600`) and the user account.
2. **Input files to parsers.** CSV, JSON, Nmap XML, and PEM files come from scanners, feeds, or other people. They are untrusted.
3. **Command line and console to the service layer.** The actor name and arguments are supplied by the user at the keyboard or in a script.
4. **The tool to targets.** Only `probe run --execute` opens connections to targets. It is the only outbound network path.

## Threats and controls

| Threat | Control in this release | Residual risk |
|---|---|---|
| Scanning an unauthorized host | Scope is required before an asset is added. Probing requires the asset to be in scope, an administrator to allow it, and `--execute`. Probing is off by default. | An authorized administrator can allow an asset that is actually out of bounds. The tool does not verify written authorization. |
| Probing too broadly or too fast | Ports capped at 64, concurrency at 16, timeouts bounded, a single connection per port. | Probing a large network with many runs is still possible. Use it with care. |
| Probing a host name that resolves elsewhere | Domain probes use the OS resolver, so a host name can resolve to another address. Scope checks the name, not the resolved address. | Known limitation. Only allow probing for names you control. |
| Malicious or malformed input files | Size limits, row and field limits, strict JSON (unknown keys refused), Nmap DTD subsets and entity declarations refused, PEM parsing limited to certificate blocks, all-or-nothing writes. Fuzz targets exist for the CSV, JSON, and Nmap parsers. | Parsers may still reject valid but unusual input. |
| SQL injection | All values are bound parameters. Migrations are static files embedded in the binary. | None known. |
| CSV formula injection in exports | Cells beginning with `=`, `+`, `-`, `@`, tab, or carriage return are prefixed with an apostrophe. | Other spreadsheet tricks outside this set are not covered. Treat exported files as data. |
| Impersonating a user | Not prevented. The actor is a name supplied by the caller. There is no password, key, or session. | **High for multi-user use.** Roles protect against mistakes, not against a hostile local user. |
| Privilege escalation through roles | Role checks run in the service layer before every operation. The last administrator cannot be demoted. The first user must be an administrator. | Anyone who can edit the database can grant any role. |
| Changing or deleting history | The audit log has database triggers that refuse UPDATE and DELETE. Each entry stores a SHA-256 hash of its content and the previous hash. `audit verify` recomputes the chain. | Someone with write access can drop the triggers and rebuild the chain. The chain detects accidental or casual edits, not a determined local attacker. |
| Secrets leaking into logs or diagnostics | There are no secrets in the tool. `diagnostics` hides the database path unless `--show-path` is given, and prints row counts rather than inventory values. Private keys in PEM files are ignored and not stored. | The database itself contains sensitive inventory data. |
| Network exposure | No listener in the main program. The lab listens on `127.0.0.1` only, with unprivileged ports and a fixed banner. | Another program on the same machine can connect to the lab while it runs. The lab serves no data. |
| Data disclosure to third parties | No telemetry, no cloud calls, no update checks. Notifications stay in the database, except that the optional notifier shows notification text on the local desktop. The notifier and GUI make no network connection. | None known for network disclosure. Anyone who can see the screen sees the text. |
| Vulnerable Go standard library | The `go` directive is `1.27.3`, the current patch release of the current Go line. CI is configured to build, test, and run govulncheck with it. On commit `0055668b`, GitHub run 40918812174 passed `govulncheck v1.9.0 (Go 1.27.3)`. That run is not for the current HEAD, and no CI job has run on HEAD. | A future standard-library vulnerability can appear before the next Go patch is adopted. Rebuild with the newer patch when it is released. See [SECURITY.md](SECURITY.md#vulnerability-scanning). |
| Denial of service on the database | SQLite with a single connection and a busy timeout of 5 seconds. | Concurrent writers from several processes can see "database is locked" errors. |
| Background notifier running unattended | Off by default. Install needs `--yes` after a consent text. Per-user autostart only, no administrator rights. Database opened read-only (`mode=ro`). Interval bounded to 1 minute to 24 hours. Default text is a count only; detailed text is capped at five messages of 120 characters. Argv-only commands, no shell. Tested with an injected sender. | A hostile local user who can write to the per-user autostart location can run code at login. This is the usual per-user autostart trust level. Real OS display and autostart are not verified on every OS. |
| GUI and file dialogs | Single local window. No listener. Exports are written only where the user chooses. Notification and bug-report links open only on click, and the bug report is previewed first. Logs, evidence, and credentials are never attached. | Anyone with access to the unlocked screen can read the inventory. |
| Release artifacts tampered with after build | Release workflow is started by hand, runs with read permissions except the final draft-creation job, refuses to overwrite an existing release, and writes SHA-256 checksums. Packaging tool is pinned to v1.7.4 and not added to go.mod. | Binaries are unsigned and not notarized, so a checksum shows only that the file matches what was uploaded. SmartScreen and Gatekeeper warnings apply. See [RELEASING.md](RELEASING.md). |
| Dependency supply chain for the GUI | Fyne v2.8.2 and its transitive modules are pinned in go.mod and go.sum. govulncheck runs in CI. No vanity-host replace directives in go.mod. | go.sum was regenerated, and `go mod tidy` and `go mod verify` passed, on a GitHub runner with Go's default proxy and checksum settings (run 40914349608, commit `a64dc88`). It was also checked locally against GitHub source archives (TESTING.md, Round 7). Transitive licenses are listed in THIRD_PARTY_NOTICES.md; the license files are copied into each release archive (RELEASING.md). |

## Out of scope for this release

- Encryption of the live database. Sealed copies made with `db seal` are encrypted, with a passphrase you keep separately. Use full-disk encryption or an encrypted volume for the live database. See [SECURITY.md](SECURITY.md#encrypted-copies).
- Authentication, sessions, or multi-user access over a network.
- Tamper-proof logging against an administrator with file access.
- Verification of ownership claims against external systems.

## Changes that require a new review

Any of these changes the model and must be reviewed before release: a network listener beyond the lab, scanning beyond TCP connect, new importers for untrusted formats, a shared or remote database, real authentication, notifications that leave the machine, or encrypted storage.

## Access control assumptions

The access model is for one trusted operator on one machine. It is a set of guardrails, not an authentication system. The assumptions are:

1. **The database file is trusted.** Anyone who can write it can add a user with any role, change a role, or edit findings. Roles are not enforced against file access.
2. **The actor name is trusted.** `--actor NAME` and `ARGUS_ASM_ACTOR` are not checked against any credential. Any process that can run the binary against the database can act as any existing user, including an administrator.
3. **The bootstrap window is trusted.** While no users exist, the actor `local` has administrator rights. Whoever creates the first user controls the database. Create the first user immediately after `init`, before the database is shared or copied.
4. **Roles guard against mistakes.** Viewer, analyst, and administrator roles stop accidental writes and make responsibility visible in the audit log. They do not stop a deliberate attacker with local access.
5. **The audit log is for the operator, not for an adversary.** Triggers stop ordinary UPDATE and DELETE statements, and the hash chain reveals casual edits. Someone who can drop the triggers and recompute the chain is not detected.

Two protections exist and are tested: the first user must be an administrator, and the last administrator cannot be demoted. Neither prevents a local attacker from acting as an administrator.

## Security implications of multi-user access

Multi-user access over a shared database file, or through a network, is not supported. If it is introduced, it changes the threat model. Before that feature is built, the following are required, and each is a separate review item:

- **Authentication.** Each user must prove identity, for example with a passphrase, a key, or the operating-system identity of the caller. The `--actor` flag would then be replaced by a verified session. The current name-only model must not be extended to a shared setting.
- **Authorization enforced at a trust boundary.** Permission checks must run where the caller cannot bypass them. Checks inside a process that the user can modify do not protect against that user.
- **Protection of the database file.** A shared file needs per-user access control, or the data must be encrypted with keys that are not stored beside it. Today the live database is not encrypted. Sealed copies are encrypted with a passphrase that the operator supplies.
- **Tamper-evident audit records.** The audit chain must be anchored somewhere the database writer cannot rewrite, or signed with a key held outside the database. A chain stored in the same file cannot prove that its author was honest.
- **Concurrency.** Several processes writing to one SQLite file can produce "database is locked" errors and must be tested. Network access would need transport security, and the spec's "no public listener" rule would need an explicit decision.
- **Least privilege for the lab and probe.** A shared installation must not let one user trigger probes against assets that another user's scope covers, unless the probe policy is per user.
- **User lifecycle.** The current model has no way to remove or disable a user. Offboarding would need a design before multi-user use.

Until these are implemented and reviewed, the tool must be used by one person per database. Do not place its database on a share that several people can write to.
