# Third-party notices

This file inventories third-party code that is compiled into or required to build argus-asm. It is maintained by hand for each release; run `go list -m all` to see the full module graph.

| Component | Version | License | Use |
|---|---|---|---|
| github.com/mattn/go-sqlite3 | v1.14.52 | MIT (Copyright (c) 2014 Yasuhiro Matsumoto) | SQLite driver (cgo) |
| SQLite (bundled inside go-sqlite3) | as bundled by v1.14.52 | Public domain (see https://sqlite.org/copyright.html) | Embedded database engine |
| fyne.io/fyne/v2 | v2.8.2 | BSD 3-Clause (Copyright (C) 2018 Fyne.io developers) | Desktop GUI toolkit, default build only (`!nogui`) |
| Lucide icons (lucide-static, pinned 1.64.0), 24 SVG files in internal/gui/icons | 1.64.0 | ISC; icons derived from Feather are MIT (Copyright (c) 2013-present Cole Bemis). Full text in internal/gui/icons/LICENSE-lucide.txt | Navigation and action icons in the GUI |
| fyne.io/tools (`cmd/fyne`) | v1.7.4 | BSD 3-Clause (Copyright (c) 2023, Fyne.io). Build-time only: run with `go run` in the release workflow, not added to go.mod and not included in any binary | Packaging the GUI into release archives (ADR 0005) |

The transitive modules of Fyne are listed in the next section. The list was produced by `go list -deps` for the default (GUI) build; run `go list -m all` for the full module graph, which also includes test-only modules.

No third-party images or fonts are included. The GUI icon set is listed above. The project does not bundle or copy code from other projects beyond the modules listed above.

## Transitive dependencies

Each module below is required by `go.mod` (directly or indirectly). The license was read from the LICENSE file of the module at the listed version, fetched from its GitHub source archive in this review. The "In default Linux build" column comes from `go list -deps ./cmd/argus-asm` with `GOOS=linux GOARCH=amd64`. For `GOOS=windows` and `GOOS=darwin` (both `GOARCH=amd64`, `CGO_ENABLED=1`) the default build contains the same modules except `github.com/rymdport/portal`, which is only included on Linux. The `nogui` build (`-tags nogui`) includes no module from this table except `mattn/go-sqlite3`.

| Module | Version | License (from its LICENSE file) | In default Linux build |
|---|---|---|---|
| fyne.io/systray | v1.12.3-0.20260810170012-af4e8e793ec4 | Apache-2.0 | yes |
| github.com/BurntSushi/toml | v1.6.0 | MIT | yes |
| github.com/FyshOS/fancyfs | v0.0.1 | BSD-3-Clause | yes |
| github.com/anthonynsimon/bild | v0.14.0 | MIT | yes |
| github.com/clipperhouse/uax29/v2 | v2.2.0 | MIT | yes |
| github.com/davecgh/go-spew | v1.1.1 | ISC | no (test or other-platform only) |
| github.com/fsnotify/fsnotify | v1.9.0 | BSD-3-Clause | yes |
| github.com/fyne-io/gl-js | v0.2.1-0.20260315212741-029c47fd27e8 | BSD-3-Clause | no (test or other-platform only) |
| github.com/fyne-io/glfw-js | v0.4.0 | MIT | no (test or other-platform only) |
| github.com/fyne-io/image | v0.1.1 | BSD-3-Clause | yes |
| github.com/fyne-io/oksvg | v0.2.1-0.20260918172519-a9af55fa95d9 | BSD-3-Clause | yes |
| github.com/go-gl/gl | v0.0.0-20260331235117-4566fea9a276 | MIT | yes |
| github.com/go-gl/glfw/v3.4/glfw | v0.1.0-pre.2 | BSD-3-Clause | yes |
| github.com/go-text/render | v0.2.1 | Unlicense OR BSD-3-Clause | yes |
| github.com/go-text/typesetting | v0.3.5 | Unlicense OR BSD-3-Clause | yes |
| github.com/godbus/dbus/v5 | v5.2.2 | BSD-2-Clause | yes |
| github.com/hack-pad/go-indexeddb | v0.3.2 | Apache-2.0 | no (test or other-platform only) |
| github.com/hack-pad/safejs | v0.1.0 | Apache-2.0 | no (test or other-platform only) |
| github.com/jeandeaual/go-locale | v0.0.0-20250612000132-0ef82f21eade | MIT | yes |
| github.com/jsummers/gobmp | v0.0.0-20230614200233-a9de23ed2e25 | MIT | yes |
| github.com/mattn/go-runewidth | v0.0.24 | MIT | yes |
| github.com/nfnt/resize | v0.0.0-20180221191011-83c6a9932646 | ISC-style permissive (as written in its LICENSE) | yes |
| github.com/nicksnyder/go-i18n/v2 | v2.5.1 | MIT | yes |
| github.com/pmezard/go-difflib | v1.0.0 | BSD-3-Clause | no (test or other-platform only) |
| github.com/rymdport/portal | v0.4.2 | Apache-2.0 | yes |
| github.com/srwiley/oksvg | v0.0.0-20221011165216-be6e8873101c | BSD-3-Clause | yes |
| github.com/srwiley/rasterx | v0.0.0-20220730225603-2ab79fcdd4ef | BSD-3-Clause | yes |
| github.com/stretchr/testify | v1.11.1 | MIT | no (test or other-platform only) |
| github.com/yuin/goldmark | v1.8.2 | MIT | yes |
| golang.org/x/image | v0.47.0 | BSD-3-Clause | yes |
| golang.org/x/net | v0.35.0 | BSD-3-Clause | yes |
| golang.org/x/sys | v0.49.0 | BSD-3-Clause | yes |
| golang.org/x/text | v0.43.0 | BSD-3-Clause | yes |
| gopkg.in/yaml.v3 | v3.0.1 | MIT OR Apache-2.0 | no (test or other-platform only) |

Notes:
- The `nogui` build includes only `mattn/go-sqlite3` from this list and no Fyne code.
- `github.com/go-gl/glfw/v3.4/glfw` v0.1.0-pre.2 has no LICENSE file at its tag archive (the tag download returned 404), so its LICENSE was read from the module's default branch, as fetched from GitHub. Its text is BSD-3-Clause.
- `gopkg.in/yaml.v3` has a NOTICE file (Apache-2.0 portion). It is not in the default Linux build, so its NOTICE is not included in binaries built here.
- The license files were fetched from GitHub source archives, not from the Go module proxy. The `go.sum` checks are in docs/TESTING.md (Round 7). A GitHub run of `go mod tidy` and `go mod verify` also passed (run 40914349608).
- Fyne itself (BSD 3-Clause) is listed above. Its license text is in the module's LICENSE file.
- The license identifiers above are the reviewer's reading of each LICENSE file. They are not a legal opinion.

