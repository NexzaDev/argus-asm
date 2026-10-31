# Contributing to argus-asm

Thank you for helping. The full developer guide is in [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md). This page is the short version.

## Ground rules

- argus-asm is a defensive, local-first tool. Changes must keep scanning limited to authorized scopes and must not add telemetry or network calls to the core workflow.
- Go source and test files contain no comments. Express intent with names, types, tests, and documentation.
- Every behavior change needs tests. Every bug fix needs a regression test.
- User-visible changes update `docs/` and `CHANGELOG.md` in the same pull request.

## Workflow

1. Open an issue first for anything larger than a small fix.
2. Branch from `main` with a descriptive name, for example `feature/csv-dedupe-report`.
3. Use Conventional Commits for messages, for example `feat(importer): reject oversized fields`.
4. Run `make check` and, for changes to parsing, `make fuzz-smoke`.
5. Open a pull request using the template and keep it focused on one change.

## Review checklist

- Tests cover normal, invalid, boundary, and failure cases.
- Errors are returned, wrapped, and actionable. None are silently dropped.
- New paths and file permissions are documented and tested where practical.
- Documentation matches the code, including flags, exit codes, and examples.
