# Releasing

This page describes how a release is built and published. Nothing here is automatic. A maintainer starts the workflow, reviews the draft, and publishes it.

## Status

- The `release.yml` workflow is written. It has not been run on GitHub in this repository. No `release.yml` run has happened. Recent CI runs on this branch have started no jobs, so no run has produced files yet. See TESTING.md.
- Nothing is signed. No code-signing or notarization certificate exists. Do not describe these files as signed or verified by a publisher.

## Files the workflow produces

Each run produces three archives and a checksum file, attached to a draft release `v<version>`:

| File | Platform | Contents |
| --- | --- | --- |
| `argus-asm-<version>-linux-<arch>.tar.xz` | Linux | Fyne package with desktop entry and icon, plus LICENSE, README.md, THIRD_PARTY_NOTICES.md, `licenses/` |
| `argus-asm-<version>-darwin-<arch>.zip` | macOS | `Argus ASM.app`, plus LICENSE, README.md, THIRD_PARTY_NOTICES.md, `licenses/` |
| `argus-asm-<version>-windows-<arch>.zip` | Windows | `argus-asm.exe`, plus LICENSE, README.md, THIRD_PARTY_NOTICES.md, `licenses/` |
| `SHA256SUMS` | all | SHA-256 of the three archives |

`<arch>` is the Go `GOARCH` of the runner (for example `amd64` or `arm64`). The file names are generated, so check the names in the draft.

The binaries are built with the GUI, so they need a desktop session to open a window. The CLI-only build (`-tags nogui`) is a separate command in COMMANDS.md and the `ci.yml` workflow.

## License files in each archive

Each build step runs `scripts/collect-licenses.sh` with the runner's own `GOOS`. The script lists the modules in the GUI build with `go list -deps`, then copies each module's LICENSE, COPYING, or NOTICE file into `licenses/<module path>/` and writes `licenses/INDEX.txt`. The step fails if a module has no license file, or if no third-party module is found. The folder is a copy of the files the module shipped with; it is not a review. The reviewed summary is THIRD_PARTY_NOTICES.md.

To run it locally on Linux or macOS: `bash scripts/collect-licenses.sh /tmp/licenses ./cmd/argus-asm`. It needs Go and a populated module cache, and it writes nothing into the repository.

## Start a release (maintainer)

1. In GitHub, open **Actions** → **Release (draft)** → **Run workflow**.
2. Enter the version without a leading `v`, for example `0.1.0`. It must match `MAJOR.MINOR.PATCH`.
3. Start the run. The `prepare` job stops the run if a release `v<version>` already exists.
4. When the run finishes, open **Releases**. The new release has a **Draft** label. It is not public.

Each run starts from the branch you select. Use `main` for a release.

## Verify before publishing

Download the three archives and `SHA256SUMS` from the draft. Check each file against the list:

- Linux or macOS: `sha256sum -c SHA256SUMS`
- Windows (PowerShell): `Get-FileHash .\argus-asm-<version>-windows-<arch>.zip -Algorithm SHA256` and compare with the line in `SHA256SUMS`

Then run the binary on a machine you control. If it does not open a window, report it. The draft does not prove that the GUI starts on each platform.

## Unsigned binaries

These files are not code-signed and not notarized. Operating systems warn about them.

- **Windows SmartScreen** may show "Windows protected your PC". Choose **More info**, then **Run anyway**. Only do this after the checksum matches.
- **macOS Gatekeeper** may refuse to open `Argus ASM.app` because the developer cannot be verified. The usual path is to open the app once from **System Settings → Privacy & Security** and choose **Open Anyway**, or to Control-click the app and choose **Open**. Apple documents these steps; they have not been tested for this app.
- **Linux** does not show a publisher warning for a tarball. Check the checksum first.

Users should only run these files after checking `SHA256SUMS`. A checksum shows the file is the one that was uploaded. It does not show who built it.

## Publish the draft from a phone

These steps use the GitHub website in a phone browser. They are based on the GitHub interface and have not been tested on a phone in this project.

1. Open the repository page in the phone browser and sign in.
2. Tap **Releases**. Drafts show a **Draft** label.
3. Tap the draft release `v<version>`.
4. Tap **Edit** (pencil icon). Read the notes. Remove any text that is not accurate.
5. Confirm the three archives and `SHA256SUMS` are attached.
6. Tap **Publish release**. Only the maintainer with write access can do this.

Publishing creates the tag `v<version>` on the commit the workflow ran against.

## Rebuild or remove a draft

- To rebuild, delete the draft release in GitHub, then run the workflow again with the same version. The tag is created only when the release is published, so a deleted draft leaves no tag behind. Check that no tag `v<version>` exists before rerunning.
- The workflow refuses to create a release that already exists. Use a new version to replace a published release.

## Dependencies used by the release build

- Go: the toolchain in `go.mod`.
- Linux graphics headers, from the Fyne getting-started guide, installed at build time: `gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev`.
- Packaging tool: `fyne.io/tools/cmd/fyne` v1.7.4, run with `go run`. It is not added to `go.mod`. See ADR 0005.

## go.sum maintenance

`go.sum` is checked against the module graph with `go mod tidy`. Run these on a machine that can reach the Go module proxy:

```
GOTOOLCHAIN=local go mod tidy
git diff --exit-code go.mod go.sum
go mod verify
```

The last two commands must print no diff and `all modules verified`.

On GitHub, the manual `go.sum refresh` workflow (`go-sum.yml`) ran `go mod tidy` and `go mod verify` successfully (run 40914349608). It committed the result as `a64dc88`. The workflow is `workflow_dispatch` only in the repository. It is still enabled on GitHub. Disabling it returned HTTP 403 to the sandbox token. A maintainer should disable it with **Actions → go.sum refresh → ⋯ → Disable workflow**. A workflow with `contents: write` should not stay enabled.

The sandbox could not reach the module proxy. Its local check used GitHub source archives instead (TESTING.md, Round 7).

## Version string inside the GUI

The About screen reads the build version from `internal/buildinfo`. The `fyne package` step sets the app version for the package metadata but does not set that variable, so the About screen can show the development version. This is a known gap.
