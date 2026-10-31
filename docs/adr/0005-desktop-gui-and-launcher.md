# ADR 0005: Desktop GUI with Fyne and an optional build without it

## Status

Accepted for milestones 1 to 5 (round 5). The packaging tool decision below supersedes the earlier one in the milestone 2 section. No third-party notification library was added; the notifier uses OS commands (see the milestone 4 section).

## Context

The specification allows an optional native desktop interface when it can be built with free tools. Fyne (v2.8.2, BSD-3-Clause) is a Go toolkit that needs cgo and the platform graphics libraries. The CLI also needs cgo for its SQLite driver, so a build without the GUI is not cgo-free.

## Decision

- The GUI is the `internal/gui` package with the `!nogui` build constraint. A build with `-tags nogui` contains no Fyne code. Verified in the repository's `nogui` CI job.
- The CLI reaches the GUI only through the `internal/desktop` interface. `cmd/argus-asm` injects the implementation. The CLI and the service layer hold no GUI code.
- The launcher modes are `gui`, `console`, and `cli`. Unset on a desktop shows a chooser. Without a display, or in a `nogui` build, the launcher starts the console and prints the reason.
- The GUI calls the same service layer as the CLI. It opens the SQLite database through the existing repository code and enforces the same roles.
- The GUI opens no network listener.

## Milestone 2 (dashboard) decisions

- Icons: Lucide, pinned lucide-static 1.64.0 (ISC; MIT for Feather-derived icons). 24 SVG files are embedded in `internal/gui/icons`. Fyne's SVG recoloring only changes fill colors, so `currentColor` is replaced with the theme foreground per render.
- Tooltips: Fyne 2.8.2 has no tooltip widget. The collapsed sidebar shows the item name in the status line on hover. This is a deliberate substitute, not a tooltip.
- Sorting uses a column selector and a direction selector, not header clicks (keeps the table simple and keyboard-accessible).
- Severity colors are chosen to meet WCAG AA (4.5:1) against Fyne's default light and dark backgrounds; `TestSeverityColorsMeetWCAGAA` checks it.
- The usage guide and third-party notices are embedded as copies of `docs/USAGE.md` and `THIRD_PARTY_NOTICES.md`. A test fails when the copies differ.
- Notification library: none added in milestone 2. Notifications use the existing service layer.
## Milestone 4 (notifier) decisions

- No third-party library was added. OS notifications run `notify-send` (Linux), `osascript` (macOS), or PowerShell with a balloon tip (Windows). Arguments are passed as argv values or environment variables, never through a shell. Real display is not verified (see TESTING.md).
- The background notifier (`argus-asm notifier run`) opens the database read-only (`mode=ro`), checks on a bounded interval (1 minute to 24 hours), and makes no network connection.
- Autostart is per-user only: Windows Startup folder, macOS LaunchAgent, Linux XDG autostart. Install needs `--yes` after the consent text is printed, and records a baseline so only later notifications are announced.
- Notification text is minimal by default. The detailed option shows at most five stored messages, each cut to 120 characters. Evidence and credentials are not used.
- Threat model: the notifier reads notification text that is already stored locally and sends it only to the local OS notification service. A hostile local user with write access to the autostart folder can run code at login. That is the same trust as any per-user autostart entry, and it is stated here rather than mitigated.

## Milestone 5 (release) decisions: packaging tool

- Decision: the release workflow runs the `fyne` packaging tool from `fyne.io/tools/cmd/fyne` at the pinned version v1.7.4, using `go run fyne.io/tools/cmd/fyne@v1.7.4 package ...`. The tool is not added to `go.mod`.
- This supersedes the milestone 2 note that chose `fyne.io/fyne/v2/cmd/fyne`. That choice was wrong for three reasons, checked against the Fyne v2.8.2 source:
  1. `fyne.io/fyne/v2/cmd/fyne` is deprecated in v2.8.2. Its `main.go` prints that the tool "has migrated to fyne.io/tools/cmd/fyne" and that it should be installed with `go install fyne.io/tools/cmd/fyne@latest`.
  2. The deprecated tool depends on `github.com/urfave/cli/v2`, which would add packages to the main module graph for a build-time tool.
  3. Its packaging commands were removed from the deprecated copy, so they cannot be checked against the source.
- Deviation from the approved wording: the approval named the main module. Following it would mean using a deprecated tool and adding its dependencies to `go.mod`. This ADR therefore uses the non-deprecated tool in its own module. The version is pinned in the workflow and recorded here.
- `go run pkg@version` builds the tool in its own module graph, so the repository's `go.mod` and `go.sum` are not changed by the release workflow.
- `fyne.io/tools` v1.7.4 requires `fyne.io/fyne/v2 v2.8.1`. The release build uses the repository's Fyne v2.8.2 for the app, and the tool uses its own v2.8.1 requirement for its own build. This is a documented mismatch and is not verified by a run in this sandbox.
- Linux packages for the GUI come from the Fyne getting-started guide: `gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev`. The release workflow installs them at build time.
- Signing: none. Binaries are unsigned and not notarized. Windows SmartScreen and macOS Gatekeeper warnings are documented in docs/RELEASING.md.

## Consequences

- The default build needs cgo and the Linux graphics headers listed in the Fyne getting-started guide. CI installs them on Linux.
- Headless GUI tests use the Fyne test driver (`-tags ci`), which does not need a display or GL headers. They do not prove that a real window opens on each OS.
- A full-screen terminal interface is still not implemented (ADR 0004).
- Release files are built on GitHub-hosted runners by a manually started workflow (`release.yml`). Nothing is signed, and nothing is published. The release is created as a draft for the maintainer to review.
