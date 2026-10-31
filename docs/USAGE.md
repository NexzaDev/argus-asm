# Usage guide

This guide walks through a complete workflow with the synthetic demo data in `testdata/demo/`. Outputs shown here were produced by the binary built from this repository. Timestamps, IDs, and counts on your machine will differ.

All names and addresses are reserved for documentation: `example.test` (RFC 2606) and `192.0.2.0/24` (RFC 5737). Do not use real third-party targets with this tool or with these examples.

## Before you start

Build the binary as described in [INSTALLATION.md](INSTALLATION.md). The examples call it as `./bin/argus-asm`. On Windows use `.\bin\argus-asm.exe`.

Use a database file of your own so the demo does not touch real data:

```sh
export ARGUS_ASM_DB="$HOME/argus-demo/argus.db"
./bin/argus-asm init
```

PowerShell: `$env:ARGUS_ASM_DB = "$HOME\argus-demo\argus.db"`. The program reads this variable from the environment. It does not read `.env` files; see [CONFIGURATION.md](CONFIGURATION.md).

## 1. Authorize scope

Nothing can be inventoried until a scope covers it. Scopes are the first control in the tool.

```sh
./bin/argus-asm scope add example.test
./bin/argus-asm scope add 192.0.2.0/24
./bin/argus-asm scope list
```

```text
added domain scope example.test
added cidr scope 192.0.2.0/24
```

A domain scope also covers its subdomains. A CIDR scope must be `/16` or narrower for IPv4.

## 2. Register assets

```sh
./bin/argus-asm asset add www.example.test
./bin/argus-asm asset add api.example.test
./bin/argus-asm asset add 192.0.2.10
./bin/argus-asm asset add 203.0.113.99
echo "exit=$?"
```

```text
added domain asset www.example.test
added domain asset api.example.test
added ip asset 192.0.2.10
error: outside every authorized scope: ip 203.0.113.99 is not inside any registered scope; add a scope first
exit=5
```

Assets outside every scope are refused with exit code 5.

## 3. Record ownership

Ownership is a record that an operator keeps. The tool stores what you enter and does not check it against an external source.

```sh
./bin/argus-asm asset owner www.example.test --owner "Web team" --verified-by "Alex Doe"
```

```text
owner of www.example.test is Web team (verified by Alex Doe)
```

## 4. Add services and certificates

Services can come from an Nmap XML file (`nmap -oX`) or be entered by hand. The demo file is synthetic.

```sh
./bin/argus-asm service import-nmap testdata/demo/nmap.xml
./bin/argus-asm cert import www.example.test testdata/demo/certificate.pem
```

```text
imported nmap services: 2 observed, 2 new, 0 updated
imported certificates for www.example.test: 1 new, 0 already known, 1 expiring within 30 days
```

The Nmap import refuses the whole file if any host in it is not in the inventory. The certificate expires within 30 days, so a notification is created.

## 5. Import findings

Check a file first with `--dry-run`, then import it. `--source` names the scanner or feed.

```sh
./bin/argus-asm findings import --source demo-scanner testdata/demo/findings.csv --dry-run
./bin/argus-asm findings import --source demo-scanner testdata/demo/findings.csv
```

```text
dry run: 6 row(s) valid; nothing written
imported 6 row(s) from demo-scanner: 6 new, 0 updated
```

A JSON feed works the same way:

```sh
./bin/argus-asm findings import --format json --source demo-json-feed testdata/demo/findings.json
```

A SARIF 2.1.0 log works too. Its assets must already be in the inventory, and its source is the tool name:

```sh
./bin/argus-asm findings import --format sarif --source demo-sarif-analyzer testdata/demo/findings.sarif --dry-run
./bin/argus-asm findings import --format sarif --source demo-sarif-analyzer testdata/demo/findings.sarif
```

Any invalid row makes the whole file fail. For example, a severity of `nope` produces:

```text
line 2: unknown severity "nope" (expected critical, high, medium, low, or info)
error: validation failed: 1 row(s) rejected; nothing was imported
```

and exit code 3.

## 6. Review by risk

```sh
./bin/argus-asm findings list --limit 3
./bin/argus-asm findings show 3
```

```text
ID  SCORE  SEVERITY  STATUS  ASSET             TITLE                                            SOURCE
3   90     critical  open    api.example.test  Admin endpoint reachable without authentication  demo-scanner
5   74     high      open    192.0.2.10        SSH service accepts password authentication      demo-scanner
4   70     high      open    api.example.test  Outdated web framework version                   demo-scanner
finding 3: Admin endpoint reachable without authentication
  asset:    api.example.test
  severity: critical   status: open   risk score: 90
  source:   demo-scanner
  ...
  risk factors:
    - severity critical contributes 90
  tickets: 0
```

Add `--explain` to `findings list` to show the factors under each row. Accepted findings stay in the default list with a score of 0, so the decision remains visible. The scoring rules are in [COMMANDS.md](COMMANDS.md#risk-score).

## 7. Decide on findings

Accepting a risk needs a reason:

```sh
./bin/argus-asm findings status 2 --set accepted
echo "exit=$?"
./bin/argus-asm findings status 2 --set accepted --reason "Compensating control: HSTS at CDN"
```

```text
error: accepting a risk requires --reason explaining the decision
exit=3
finding 2 is now accepted
```


## 8. Track remediation

```sh
./bin/argus-asm ticket create --finding 3 --owner platform-team --due 2026-11-05
./bin/argus-asm ticket list
./bin/argus-asm notify list
```

```text
created ticket 1 for finding 3 (owner platform-team, due 2026-11-05)
ID  STATUS  DUE         OWNER          ASSET             FINDING
1   open    2026-11-05  platform-team  api.example.test  Admin endpoint reachable without authentication
```

Imports create a notification for each critical or high finding, and certificate imports create one for a certificate that expires within 30 days. `notify refresh` adds notifications for overdue tickets and for expiring certificates. Notifications are stored in the database. `notify deliver --to FILE` can write the undelivered ones to a local file (see [Notification files and sealed copies](#notification-files-and-sealed-copies)). `notify ack ID` marks one as read.

## 9. Compare imports

Change detection compares the two most recent imports from the same source. The first import of a source cannot be compared, so a second import is needed. Here the demo file was edited (one finding removed, one severity lowered, one added) and imported again:

```sh
./bin/argus-asm findings import --source demo-scanner testdata/demo/findings-second.csv
./bin/argus-asm findings changes --source demo-scanner
```

```text
imported 6 row(s) from demo-scanner: 1 new, 5 updated
changes for demo-scanner between batch 1 and batch 4
  new: 1   missing: 1   severity changed: 1
  + [medium] www.example.test: Newly observed debug endpoint
  - [high] api.example.test: Outdated web framework version
  ~ 192.0.2.10: Banner discloses server software version info -> low
```

`+` is a new finding, `-` is a finding missing from the latest import, and `~` is a severity change. Batch numbers count every import, so this walkthrough's second CSV is batch 4, after the JSON and SARIF imports in section 5. `findings batches --source NAME` lists the imports that can be compared.

## 10. Export

```sh
./bin/argus-asm export findings --format csv --output findings.csv
./bin/argus-asm export assets --format json
```

Export is limited to users with the analyst role. CSV cells that could be read as formulas by a spreadsheet are prefixed with an apostrophe.

## 11. Active checks (optional)

Active checks are off by default. Probing an asset needs three things: the asset is in a scope, an administrator has allowed probing for it, and you run `--execute`. Without `--execute` the command only prints a plan.

```sh
./bin/argus-asm asset probe allow 192.0.2.10
./bin/argus-asm probe run 192.0.2.10 --ports 22,443
```

```text
active probing of 192.0.2.10 is now allowed
plan for 192.0.2.10: 2 TCP port(s) 22,443; nothing was contacted (add --execute to run)
```

To try a real connection without touching any remote host, use the local lab in a second terminal:

```sh
./bin/argus-asm scope add 127.0.0.0/16
./bin/argus-asm asset add 127.0.0.1
./bin/argus-asm asset probe allow 127.0.0.1
./bin/argus-asm lab serve
```

Then in the first terminal:

```sh
./bin/argus-asm probe run 127.0.0.1 --ports 18080,18022,19999 --execute --timeout 1s
```

```text
probed 127.0.0.1: 2 open, 1 closed or filtered
  open 18022/tcp
  open 18080/tcp
```

Port 19999 is closed in this example because the lab does not listen on it. Each open port is recorded as a service with source `tcp-connect`, so `service list --asset 127.0.0.1` shows them. Probing never changes a finding.

## Notification files and sealed copies

These commands write files outside the database. Keep the files out of version control.

### Notification file

`notify deliver --to FILE` appends each undelivered notification to `FILE` as one JSON object per line. The first run marks them delivered, so a second run writes nothing:

```sh
./bin/argus-asm notify deliver --to ./outbox.jsonl --dry-run --actor root
./bin/argus-asm notify deliver --to ./outbox.jsonl --actor root
./bin/argus-asm notify deliver --to ./outbox.jsonl --actor root
```

Delivery is at-least-once. If the command stops after writing but before marking, the next run writes those lines again, so use the `id` field to skip duplicates. The command sends nothing over a network. See [COMMANDS.md](COMMANDS.md#notify-deliver---to-file---dry-run).

### Encrypted copy

An administrator can write an encrypted copy of the database. The passphrase comes only from standard input and must be at least 12 bytes. The bash pattern below keeps it out of the command line and shell history:

```sh
read -rs PASS
printf '%s\n' "$PASS" | ./bin/argus-asm db seal --out ./argus-backup.sealed --passphrase-stdin --actor root
unset PASS
```

To restore the copy into a new file and check it:

```sh
read -rs PASS
printf '%s\n' "$PASS" | ./bin/argus-asm db unseal --in ./argus-backup.sealed --out ./restored.db --passphrase-stdin
unset PASS
ARGUS_ASM_DB=./restored.db ./bin/argus-asm audit verify --actor root
```

Keep the passphrase apart from the sealed file. A lost passphrase cannot be recovered. The live database is still plaintext while argus-asm uses it. See [SECURITY.md](SECURITY.md#encrypted-copies). The PowerShell form is not documented, because the pattern above has not been tested on Windows.

## Roles and actors

Access control in this release is role-based, and the actor is named on each command with `--actor NAME` or the `ARGUS_ASM_ACTOR` variable.

| Role | Can do |
|---|---|
| viewer | Read lists, search, findings, changes, and notifications |
| analyst | Everything a viewer can, plus importing, owners, links, services, certificates, findings status, tickets, notification acknowledgement, notification delivery to a file, probing (when allowed), and export |
| admin | Everything an analyst can, plus scopes, removing assets, probe permission, users, the audit log, and sealed copies (`db seal`). `db unseal` needs no actor, because it only writes a new file. |

Until a user exists, the built-in actor `local` acts as an administrator. The first user must be an administrator. After that, every command needs a known actor:

```sh
./bin/argus-asm user add root --role admin
./bin/argus-asm user add vera --role viewer --actor root
./bin/argus-asm findings list --actor vera --limit 2
./bin/argus-asm findings list
```

```text
added user vera with role viewer
ID  SCORE  SEVERITY  STATUS  ASSET             TITLE                                            SOURCE
3   90     critical  open    api.example.test  Admin endpoint reachable without authentication  demo-scanner
5   74     high      open    192.0.2.10        SSH service accepts password authentication      demo-scanner
error: permission denied: an actor is required; pass --actor or set ARGUS_ASM_ACTOR
```

Exit code 6 means permission was denied.

**Important:** the actor name is not a login. It is not checked against a password or key. Anyone who can run the binary against the same database can name any user. Roles guard against mistakes and document responsibility; they are not a security boundary. See [THREAT_MODEL.md](THREAT_MODEL.md).

## Desktop dashboard

Start the dashboard with `argus-asm --ui gui`, or pick **Dashboard (graphical window)** in the launcher chooser. It needs a graphical display. Without one, the launcher prints the reason and starts the terminal console.

- The sidebar lists Overview, Scope, Assets, Findings, Imports, Changes, Notifications, Access, Audit, Reports, Settings, and About. The collapse button (or Ctrl+B) shows icons only; the name of the item under the pointer appears in the status line at the bottom.
- Shortcuts: Ctrl+1 to Ctrl+9 and Ctrl+0 open the first ten sidebar items, Ctrl+, opens Settings, F1 opens About, Ctrl+F focuses the table filter, Ctrl+R reloads stored data. On macOS, Ctrl means Cmd.
- The header says `Demo data` when stored assets use documentation-only names (`.test`, `.example`, `.invalid`, `example.com`, or 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24).
- Tables: type in the filter, choose a sort column and direction, select a row to see its details, **Copy row**, **Export CSV** (cells that start with spreadsheet formula characters are neutralized), and density switch.
- Overview: open findings by risk tier (Critical 90 and above, High 70 to 89, Medium 40 to 69, Low below 40 on the risk score), top open findings with the score explanation, assets outside every scope or without a recent finding, the last import, and the change since the previous import from the same source.
- Notifications: the bell at the bottom right shows the unread count and opens this category. Filter by **Unread only** or **All notifications**, and by severity (All, Critical, High, Medium, Low, Info, or Not rated). Each row shows its severity as an icon, text, and color. **Mark as read** acknowledges the selected notification. The window does not create notifications; run `argus-asm notify refresh` in a terminal for that.
- Reports: CSV exports are written only to the path you choose.
- Settings: table density, forget the remembered launcher choice, reload data. Colors follow the operating system light or dark setting.
- About: Overview (with **Source Code** and **Found a bug? Please report it on GitHub.**), Usage (search the text of this guide), Shortcuts, Diagnostics (preview and copy; no paths except the database file name), and Licenses. The bug link shows a preview first and opens only `https://github.com/NexzaDev/argus-asm/issues/new` in your browser after you confirm. Logs, evidence, and credentials are never attached.

External links open only when you click them, and only to the project URL above.

## Background notifier (optional)

The notifier shows an operating-system notification when new notifications are stored. It is off by default. Nothing starts until you run `notifier install --yes`.

```sh
./bin/argus-asm notifier status
./bin/argus-asm notifier install --yes --interval 5m --detail minimal
./bin/argus-asm notifier uninstall
```

- It opens the database read-only and checks on the interval you choose (1 minute to 24 hours). It makes no network connection and needs no administrator rights.
- Autostart is per user: the Windows Startup folder, a macOS LaunchAgent, or an XDG autostart entry on Linux. `notifier status` shows where.
- Install records the current notifications as already seen. Only notifications created after installation are announced.
- The default text is a count, for example "2 new critical findings in Argus". `--detail detailed` adds up to five stored messages, each cut to 120 characters. Evidence and credentials are never included.
- The notification tools are `notify-send` on Linux, `osascript` on macOS, and PowerShell on Windows. If the tool is missing, `notifier status` says so and nothing is shown.
- Whether notifications appear on your desktop has not been verified on every operating system. See [TESTING.md](TESTING.md).

See [COMMANDS.md](COMMANDS.md#background-notifier-notifier-runinstalluninstallstatus) for the full reference.

## Console

For a session with many commands, start the console:

```sh
./bin/argus-asm console --actor root
```

```text
argus-asm> findings list --limit 2
argus-asm> ticket list --overdue
argus-asm> exit
```

Each line is one command from [COMMANDS.md](COMMANDS.md). A failing command reports its error and the session continues. The console reads plain lines, so it also works with piped input:

```sh
printf 'findings list --limit 2\nexit\n' | ./bin/argus-asm console --actor root
```

## Audit log

Every change is recorded in an audit log. Administrators can read and verify it:

```sh
./bin/argus-asm audit list --actor root
./bin/argus-asm audit verify --actor root
```

```text
audit chain intact: <N> chained entries verified, <M> legacy entries without hashes
```

The counts depend on your database. A broken chain exits with code 3.

## Cleaning up

Remove an asset and see what it would delete first:

```sh
./bin/argus-asm asset remove 192.0.2.10 --dry-run --actor root
./bin/argus-asm asset remove 192.0.2.10 --actor root
```

To reset the demo, delete the database file. Nothing else is written outside it. See [INSTALLATION.md](INSTALLATION.md#uninstall-and-reset).
