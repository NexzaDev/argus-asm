package cli

import (
	"fmt"
	"io"
)

const usageText = `argus-asm - local, defensive attack surface inventory and findings tracker

Usage:
  argus-asm <command> [arguments] [flags]

Every command except init, version, about, help, and diagnostics accepts
--actor NAME (or ARGUS_ASM_ACTOR). Before any user exists, the bootstrap actor
"local" has administrator rights. Roles: viewer (read), analyst (operate),
admin (administer).

Setup:
  init                              Create or migrate the local SQLite database
  diagnostics                       Print a sanitized environment and database report
  version | about | help            Version, project information, usage

Scope and inventory (authorized scope only):
  scope add <domain|cidr>           Authorize a domain (and subdomains) or CIDR range
  scope list
  asset add <domain|ip>             Register an asset inside an existing scope
  asset list | show <asset>
  asset remove <asset> [--dry-run]  Remove an asset and report what would be deleted
  asset owner <asset> --owner NAME [--verified-by NAME]
  asset link <from> <to> --kind resolves_to|hosts|related_to
  asset links <asset>
  asset probe allow|deny <asset>    Set the per-asset active-probe permission

Services and certificates:
  service add <asset> <port[/tcp|udp]> [--name NAME]
  service list [--asset ASSET]
  service import-nmap <file|->      Import open ports from Nmap XML (-oX output)
  cert import <asset> <pem-file>    Import PEM certificates seen on an asset
  cert list

Active checks (off by default, TCP connect only):
  probe run <asset> --ports 22,443 [--execute] [--timeout 2s] [--concurrency 4]

Local lab (synthetic loopback target for trying probes safely):
  lab serve [--ports 18080,18022]   Listen on 127.0.0.1 only until Ctrl-C

Findings and lifecycle:
  findings import --format csv|json|sarif --source NAME <file|-> [--dry-run]
  findings list [--status S] [--severity V] [--asset TEXT] [--all] [--limit N] [--json]
  findings show <id>
  findings status <id> --set open|accepted|resolved [--reason TEXT]
  findings batches [--source NAME]
  findings changes --source NAME    Compare the two most recent imports from a source
  ticket create --finding ID --owner NAME --due YYYY-MM-DD
  ticket list [--overdue]
  ticket status <id> --set open|in_progress|done

Search, notifications, export, and audit:
  search <term>
  notify list [--all] | ack <id> | refresh | deliver --to FILE [--dry-run]
  export findings|assets --format csv|json [--all] [--output FILE]
  audit list | verify               Administrator-only hash-chain report

Users:
  user add <name> --role viewer|analyst|admin
  user list
  user role <name> --role viewer|analyst|admin

Database sealing (encryption at rest for copies, not for the open database):
  db seal --out FILE --passphrase-stdin      Administrator-only encrypted snapshot
  db unseal --in FILE --out FILE --passphrase-stdin
                                             Decrypt to a new database file

Console:
  console [--db PATH] [--actor NAME]  Line-based shell; each line runs one command above.
                                    Read input from files inside the console (no '-').

Launcher and preferences:
  argus-asm (no command)            Desktop: a chooser (Dashboard or Terminal). Without a display: the console.
  --ui gui|console|cli              Choose the launcher mode for a bare run. cli prints usage.
  config show                       Show the stored or environment UI mode and its source
  config set-ui gui|console|cli     Store the UI preference in the user config directory
  config reset-ui                   Remove the stored UI preference
  Precedence: --ui, then ARGUS_UI, then the stored preference, then the chooser or console.

Background notifier (off by default, per-user, no network):
  notifier run [--interval 5m] [--detail minimal|detailed]   Check the database read-only and show OS notifications
  notifier install --yes [--interval ..] [--detail ..]       Add a per-user autostart entry (Startup folder, LaunchAgent, or XDG autostart)
  notifier uninstall                                        Remove the autostart entry
  notifier status                                           Show autostart state and notification support

Global behavior:
  --db PATH overrides ARGUS_ASM_DB, which overrides the per-user default location.
  Run 'argus-asm <command> -h' for command flags.

Exit codes:
  0 success   1 failure   2 usage error   3 invalid input or not found
  4 database not initialized   5 target outside authorized scope   6 permission denied
`

func printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}
