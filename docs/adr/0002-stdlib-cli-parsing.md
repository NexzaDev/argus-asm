# ADR 0002: Standard library for the command-line interface

## Status

Accepted.

## Context

The CLI has a small, fixed command set. Third-party command frameworks add dependencies that must be licensed, reviewed, and kept current.

## Decision

Use the standard `flag` package with a small dispatcher in `internal/cli`. Flags may appear before or after positional arguments, which the dispatcher handles explicitly.

## Consequences

- Positive: no additional dependency, predictable behavior, and simple tests.
- Negative: generated shell completion and nested help are not available. Add them only if the command set grows enough to justify a dependency.
