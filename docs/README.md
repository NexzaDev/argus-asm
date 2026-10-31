# argus-asm documentation

argus-asm is a local, defensive command-line tool for recording authorized assets and tracking findings in a SQLite database. Start with the root [README](../README.md) for an overview.

## Audience

- **Operators** running the tool on their own computer: [INSTALLATION.md](INSTALLATION.md), [USAGE.md](USAGE.md), [COMMANDS.md](COMMANDS.md), [CONFIGURATION.md](CONFIGURATION.md), [TROUBLESHOOTING.md](TROUBLESHOOTING.md).
- **Security reviewers**: [SECURITY.md](SECURITY.md), [THREAT_MODEL.md](THREAT_MODEL.md), [PLATFORM_SUPPORT.md](PLATFORM_SUPPORT.md).
- **Contributors**: [CONTRIBUTING.md](CONTRIBUTING.md), [TESTING.md](TESTING.md), [ARCHITECTURE.md](ARCHITECTURE.md), and the decision records in [adr/](adr/).
- **Planning**: [ROADMAP.md](ROADMAP.md).

## Canonical references

- Command reference: [COMMANDS.md](COMMANDS.md)
- Configuration reference: [CONFIGURATION.md](CONFIGURATION.md)
- Original product specification: [SPECIFICATION.md](SPECIFICATION.md). It describes the intended full product. Behavior that is not implemented is listed in [ROADMAP.md](ROADMAP.md), not in the operational docs.

## Status

The current release line is `0.1.0-dev`. It implements scopes, inventory, services, certificates, Nmap, CSV, JSON, and SARIF importers, risk-ordered findings, lifecycle decisions, tickets, change detection, notifications (stored locally and deliverable to a file you choose), search, export, encrypted sealed copies of the database, roles, an audit log, an allowlisted TCP connect probe, a loopback lab, and a line-based console. See [ROADMAP.md](ROADMAP.md) for what is not implemented.
