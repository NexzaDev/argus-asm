# ADR 0004: Line-based console instead of a full-screen TUI

## Status

Accepted.

## Context

The specification lists a terminal user interface. A full-screen terminal interface needs raw terminal mode and cursor control, which behave differently on Linux, macOS, and Windows. Those differences cannot be verified for this release on every platform.

## Decision

The console (`argus-asm console`) reads one command per line from standard input and runs it through the same parser as the command line. It prints the same output. A failing line reports its error and the session continues. `exit` or end of input stops it. The console refuses `-` (standard input as an import source), refuses to start another console,.

## Consequences

- The workflow works the same in a terminal, in a script, and with piped input.
- There is no panel layout, history, or keyboard navigation. These are not implemented.
- A full-screen interface can be added later as a separate package that reuses the service layer, after a platform test plan exists.

## Amendment (round 5)

A desktop window was added in round 5 as a separate interface, not a full-screen terminal interface. The line-based console remains the terminal interface. See [ADR 0005](0005-desktop-gui-and-launcher.md).
