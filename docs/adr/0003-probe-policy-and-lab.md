# ADR 0003: Active-probe policy and the loopback lab

## Status

Accepted.

## Context

The specification asks for allowlisted active checks and a local lab, and forbids scanning third-party targets by default. Active checks are the only feature that sends traffic to a host, so their rules must be strict and easy to read.

## Decision

- Probing is off by default. Each asset has a probe permission, which is off until an administrator runs `asset probe allow`.
- A probe requires the `operate` role, an asset that is in scope at the time of the run, and the permission above.
- `probe run` prints a plan and sends nothing unless `--execute` is given.
- The probe performs TCP connect only, on at most 64 ports, with at most 16 concurrent attempts and timeouts between 100ms and 10s. It does not send payloads, read banners, or use UDP.
- Open ports are stored as services with source `tcp-connect`. Findings are never created or changed by a probe.
- The lab (`lab serve`) listens on `127.0.0.1` only, on ports 1024 to 65535, at most 8 at a time. It writes one fixed line per connection and closes it. It runs only while the command runs.
- The main program has no listener.

## Consequences

- Users can exercise the probe against the lab without contacting any remote host.
- Scope is checked from the declared asset value. A domain name can resolve to an address outside the scope, so administrators must allow probing only for names they control. This is recorded in the threat model.
- Adding protocols, payloads, or listeners later requires a new ADR and a threat-model update.
