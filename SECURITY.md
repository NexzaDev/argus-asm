# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| `main` | Yes |
| `v0.1.0-dev` (pre-release) | Yes |

While the project is in early development, only the latest `main` and the latest pre-release receive security fixes.

## Reporting a vulnerability

Do not open a public issue or post vulnerability details anywhere public.

### Preferred channel: GitHub private vulnerability reporting

Open the **Security** tab of the repository and choose **Report a vulnerability**, or go directly to https://github.com/NexzaDev/argus-asm/security/advisories/new.

### Backup channel: email

If you cannot use the GitHub form, send the report to **SuperMcDuck@proton.me**.

## What to include

- The affected version or commit, and your operating system and architecture
- Steps to reproduce, using synthetic data where possible
- The impact you observed, such as data exposure, unauthorized scanning, file permission problems, or a crash on crafted input

## Response expectations

The maintainer aims to acknowledge a report within 7 days and to share an initial assessment within 14 days. These are targets for a volunteer-run project, not guarantees.

## Scope

In scope: the `argus-asm` binary, its CSV, JSON, SARIF, and Nmap XML parsers, its SQLite storage and migrations, scope enforcement, file permissions it sets, the sealed-copy format, and any network behavior of the program.

Out of scope: vulnerabilities in third-party targets you do not own or have no written authorization to test, and issues that require a user to run modified code.

## Disclosure

Please allow time for a fix before public disclosure. Credit is given in the changelog on request.

See `docs/SECURITY.md` for the security model, the sealed-copy model, the scanning results, and operational guidance.
