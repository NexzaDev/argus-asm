# Contributing

This document is the full developer guide. It must agree with the root [CONTRIBUTING.md](../CONTRIBUTING.md).

## Local setup

1. Install Go 1.27.3 (the version in `go.mod`) and a C compiler (see [INSTALLATION.md](INSTALLATION.md)).
2. Clone the repository and run `make check`.
3. Without `make`, run the commands in [TESTING.md](TESTING.md).

## Code rules

- **No comments** in Go source or test files. Not inline, not block, not commented-out code. Express intent with names, types, tests, and documentation under `docs/`.
- Format with `gofmt`. Keep imports grouped by the standard formatter.
- Return errors with context (`fmt.Errorf("...: %w", err)`). Do not discard errors. Use sentinel errors from `internal/domain` for conditions the CLI must map to exit codes.
- Keep business rules in `internal/domain` or `internal/service`. The CLI parses, calls, and prints. The console and any future interface must reuse the service layer and must not duplicate rules.
- Add a dependency only with a concrete need. Review its license and update `THIRD_PARTY_NOTICES.md` and the ADR list if the choice is significant.
- No placeholder functions, stubbed return values, or buttons that do nothing. Unimplemented features belong in `ROADMAP.md`.

## Tests

- Every behavior change has a test. Every bug fix has a regression test that fails before the fix.
- Tests must be deterministic: no real network, no real hostnames or addresses (use `example.test`, `192.0.2.0/24`, and similar reserved names), no dependence on the current time (inject a clock into `service.New`).
- Use `t.TempDir()` for databases and files.
- Parsers that accept external input get a fuzz target. Run `make fuzz-smoke` after changing them.

## Documentation

- Update the relevant file in `docs/` in the same pull request as the code change. Examples must be run against the built binary before they are committed.
- Flags, exit codes, configuration names, and file formats must match the code exactly.
- Mark anything planned as planned. Do not describe unimplemented behavior as available.

## Commits and branches

- Branch from `main`. Use short prefixes such as `feature/`, `fix/`, `docs/`, or `chore/`.
- Use Conventional Commits: `type(scope): summary`, with types such as `feat`, `fix`, `docs`, `test`, `refactor`, `build`, `ci`, and `chore`.
- Keep commits focused. A pull request should describe one change.

## Pull request review checklist

- [ ] `make check` passes, or the equivalent commands are listed with their results.
- [ ] Tests cover normal, invalid, boundary, and failure cases for the change.
- [ ] No new comments in Go source or tests.
- [ ] No new network calls, telemetry, or scanning behavior. If any is proposed, the threat model is updated first.
- [ ] Scope enforcement and import validation are unchanged, or changed with tests.
- [ ] Docs and `CHANGELOG.md` are updated.
- [ ] No real data, secrets, or personal information in code, fixtures, or logs.

## Security-sensitive changes

Changes to parsing of external input, file permissions, scope rules, SQL, or any network behavior require a review of [THREAT_MODEL.md](THREAT_MODEL.md) and [SECURITY.md](SECURITY.md) in the same pull request. Report vulnerabilities privately as described in the root `SECURITY.md`, not in a pull request.
