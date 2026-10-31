# Argus ASM — Attack Surface Management Platform

**Repository:** `argus-asm`

**GitHub description:** A defensive platform for authorized asset inventory, domain/subdomain tracking, service exposure records, vulnerability imports, risk prioritization, change history, and remediation workflows.

## Language stack

| Layer | Choice | Responsibility |
|---|---|---|
| Core/backend | Go | Domain logic, performance-sensitive work, APIs, integrations, and background tasks |
| Interface | Go CLI/TUI + optional Fyne desktop GUI | Local terminal UI; optional native desktop GUI; no browser frontend |
| Persistence | SQLite | Durable application data and queryable history |

Go owns API, asset inventory, scheduling, integrations, and risk calculations; the interface is a local terminal UI. Use Python only for optional offline data-analysis utilities, not the main service.

## What this repository is

This README contains the project specification and the complete prompt intended for an AI coding agent. It does not claim the application has already been implemented.

## Product scope

Build an enterprise-style ASM platform with authorized asset scopes, verified ownership records, domain/subdomain/IP/certificate/service inventory, passive discovery where practical, allowlisted active checks, importers for scanner JSON/CSV, deduplication, asset relationships, explainable risk scoring, findings lifecycle, remediation tickets, change detection, notifications, RBAC, search, exports, and a safe local lab. Never scan arbitrary third-party targets by default.

## Non-negotiable delivery constraints

- This is **not a website, web app, SaaS product, or hosted dashboard**. Do not create a browser frontend, React app, or public-facing web server.
- Deliver a local-first **CLI/TUI application** and, where useful, a polished cross-platform native desktop GUI. No website or browser frontend. Any optional listener must be disabled by default and bind to loopback only unless explicitly configured.
- The complete core workflow must run for **€0** on a normal developer computer. Use free/open-source tools and libraries, with no required paid API keys, subscriptions, cloud accounts, hosted databases, or commercial services.
- Default to local files and SQLite where persistence is needed. External integrations must be optional adapters, never required for core functionality; provide fixtures and a fully usable offline/demo mode.
- Do not require paid CI, hosting, domains, premium IDEs, or paid threat-intelligence feeds. Document optional free-tier limits honestly and never make a free-tier account mandatory.
- Use a free, OS-supported build/test workflow. State the exact local prerequisites and commands needed to build, test, and run the project.

## Cross-platform and product-quality requirements

- **Supported operating systems:** Linux, macOS, and Windows. The project must have documented build, test, and run instructions for all three, including architecture-specific notes where relevant.
- **Fast, suitable implementation languages:** Go is the default for network tools, collectors, services, and concurrent workloads; Rust is preferred for memory-sensitive, parsing-heavy, or performance-critical components. Do not use Python as the main implementation language. Use it only for optional developer scripts if there is a clear reason.
- **No website:** Do not create a website, hosted dashboard, SaaS product, browser frontend, or public web server. This is a local application/tool.
- **Professional interface:** Provide a polished CLI with consistent command naming, useful help, clear errors, structured output, progress indicators, and accessible color handling. Add a native desktop GUI when it materially improves the workflow; use a cross-platform native UI toolkit (Fyne for Go or egui/Slint for Rust), not a browser-based UI. Visual tools must have a real interface, not just a placeholder screen.
- **Distinctive, production-quality UX:** Use a coherent design system, deliberate spacing and typography, restrained colors, meaningful empty/loading/error/success states, keyboard navigation, sensible defaults, accessible contrast, responsive layouts, and realistic sample data. Avoid generic AI-generated dashboard templates, gratuitous gradients, meaningless charts, fake metrics, placeholder buttons, lorem ipsum, and decorative features that do not work.
- **Real functionality only:** Every visible control must work. Use representative local fixtures for demos; label demo data clearly. Do not invent integrations, results, benchmarks, or completed tests.
- **Cross-platform verification:** CI must build and test on Linux, macOS, and Windows where supported. Use portable filesystem paths, signal handling, process management, permissions, and networking APIs. Test path handling and platform-specific behavior. Do not claim a platform is supported until its build/test path is documented and verified.
- **€0 operation:** Core functionality must run locally using free/open-source dependencies without paid API keys, subscriptions, cloud accounts, hosted databases, domains, or commercial services. External integrations are optional and must have an offline/demo alternative.
- **Performance discipline:** Profile before optimizing; use bounded concurrency, cancellation, timeouts, streaming for large inputs, and benchmarks for genuinely hot paths. Avoid unnecessary allocations and heavyweight dependencies, but do not trade correctness or maintainability for synthetic benchmark numbers.
- **No source-code comments:** Keep source files free of comments as requested. Express design intent through clear names, small cohesive modules, tests, and documentation outside source files.
- **No AI slop:** Do not fabricate features or inflate scope/LOC. Prefer a small set of complete, tested workflows over many incomplete screens. Use consistent naming, idiomatic language patterns, explicit error handling, deterministic tests, dependency/license review, and a clean release process.

## Product polish, branding, and support features

- **Custom icon and identity:** If the application has an icon, create an original project-specific icon instead of a generic placeholder. Provide editable SVG source artwork plus platform assets as applicable (Windows `.ico`, macOS `.icns`, Linux SVG/PNG and desktop-entry assets). Do not use third-party trademarks.
- **Source access:** GUI applications must have a visible **Source Code** action in About that opens the real public GitHub repository URL configured for the project. Keep the URL in one central metadata/config location. If it is not configured, show an honest fallback rather than a fabricated URL. CLI-only tools must provide an `about` command with version, license, project URL when configured, OS and build details.
- **About page/dialog:** Every GUI application must include an About screen/dialog showing app name, icon, version, purpose, license, maintainer/copyright only when known, repository link, dependency notices, and build/runtime information where useful. Include a **“Found a bug? Please report it on GitHub.”** action that opens the repository issue tracker when configured. Never fabricate repository URLs.
- **Support and diagnostics:** Include a previewable, copyable diagnostic report with app version, OS, architecture, runtime/build versions and sanitized configuration status. Do not include credentials, tokens, personal file contents, or sensitive host/network data by default.
- **Useful product features:** Contextual help, searchable help for complex tools, keyboard shortcuts, copy/export actions, safe settings defaults, first-run guidance, actionable errors, cancellation, confirmation for destructive actions, and clear empty/loading/error/success states.
- **Accessibility:** Keyboard navigation, readable contrast, scalable text, visible focus indicators, and screen-reader labels where supported. Do not rely on color alone to communicate severity.
- **Security and privacy:** No telemetry by default. Disclose external network requests, redact secrets from logs, use OS-appropriate config directories and restrictive permissions, and provide dry-run mode for potentially disruptive operations.
- **Release quality:** Versioning, changelog, checksums and signed release artifacts where feasible with free tools, reproducible build guidance, dependency/license inventory, uninstall/reset instructions, and platform-specific troubleshooting.
- **Honest completeness:** No fake buttons, fake charts, fabricated performance numbers, placeholder links, or screens disconnected from real functionality. Label demo data clearly and omit features that are not implemented.

## Engineering quality, correctness, and release gates

- **No promise of zero bugs:** Software cannot be guaranteed bug-free. The engineering process must systematically reduce defects, catch regressions, and document known limitations instead of claiming perfection.
- **Requirements before implementation:** Convert every feature into acceptance criteria and test cases. Keep a traceable checklist linking requirements to code, tests, documentation, and release status.
- **Small, complete increments:** Implement vertical slices that compile and work end-to-end. Do not generate the whole application in one unverified pass or leave placeholder modules and stubbed workflows.
- **Compile and test continuously:** After each meaningful change, run formatter, compiler, static analysis/linter, unit tests, integration tests, and the relevant platform build. Fix failures before moving forward; never suppress a failure just to get a green result.
- **Defensive engineering:** Validate all external input, handle malformed files and partial network failures, set explicit timeouts, cancel work cleanly, bound concurrency and memory, avoid races, and return actionable errors. Never silently swallow errors.
- **Test design:** Cover normal behavior, boundary cases, invalid input, empty input, large inputs, permission failures, cancellation, timeouts, corrupt state, offline mode, and regression cases. Use deterministic tests, fixtures, fuzz/property testing where appropriate, and race/concurrency checks for Go or Rust tools as applicable.
- **Cross-platform correctness:** Build and test on Linux, macOS, and Windows. Test paths, permissions, signals, line endings, process handling, sockets, and packaging on the actual target platforms. If a platform cannot be tested, disclose it instead of claiming full verification.
- **Security assurance:** Perform threat modeling, dependency vulnerability review, secret scanning, static analysis, safe parsing, least-privilege design, secure defaults, and checks for path traversal, command injection, unsafe deserialization, data leakage, and unintended network exposure.
- **Performance validation:** Set realistic performance goals and benchmark representative workloads with reproducible commands and hardware/context notes. Profile bottlenecks before optimization. Never invent benchmark numbers or sacrifice correctness for synthetic speed.
- **Code review and maintainability:** Use idiomatic Go/Rust, clear module boundaries, explicit ownership and error contracts, minimal dependencies, no dead code, no duplicated logic, no unexplained magic values, and no artificial LOC padding. Source code must contain no comments as requested; use descriptive names, tests, and external docs.
- **Release blocking gates:** A release must not be marked ready if required builds/tests/lints/security checks fail, core flows are untested, documentation is inconsistent, or critical/high-severity known defects remain unresolved without explicit risk acceptance.
- **Documentation acceptance gate:** The implementation must contain a non-empty `docs/` directory with all required files: `README.md`, `INSTALLATION.md`, `USAGE.md`, `CONFIGURATION.md`, `COMMANDS.md`, `ARCHITECTURE.md`, `TROUBLESHOOTING.md`, `SECURITY.md`, `TESTING.md`, `CONTRIBUTING.md`, `PLATFORM_SUPPORT.md`, `THREAT_MODEL.md`, and `ROADMAP.md`. Every important feature, command, flag, configuration option, data format, permission, integration, error/recovery path, and platform-specific behavior must be documented. Examples must be checked against the actual application; never invent output or claim unsupported platform compatibility. Update docs in the same change as code, and block release when docs are stale, missing, contradictory, or materially incomplete.
- **Honest reporting:** Report exactly which commands ran, which platforms were tested, what passed/failed, any skipped checks, known issues, and remaining risks. Never claim a test, scan, benchmark, or build ran unless it actually ran.

## Proposed codebase layout

```text
argus-asm/
├── README.md
├── LICENSE
├── CONTRIBUTING.md
├── SECURITY.md
├── CODE_OF_CONDUCT.md
├── CHANGELOG.md
├── .editorconfig
├── .gitignore
├── .env.example
├── Makefile
├── cmd/
│   └── argus-asm/
├── internal/
│   ├── config/
│   ├── domain/
│   ├── service/
│   ├── repository/
│   ├── transport/
│   └── observability/
├── api/
├── tui/
│   ├── views/
│   ├── components/
│   ├── formatting/
│   └── tests/
├── migrations/
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
├── testdata/
├── scripts/
├── assets/
│   ├── icons/
│   │   ├── icon.svg
│   │   ├── icon.png
│   │   ├── icon.ico
│   │   └── icon.icns
│   └── branding/
├── docs/
│   ├── README.md
│   ├── INSTALLATION.md
│   ├── USAGE.md
│   ├── CONFIGURATION.md
│   ├── COMMANDS.md
│   ├── ARCHITECTURE.md
│   ├── TROUBLESHOOTING.md
│   ├── SECURITY.md
│   ├── TESTING.md
│   ├── CONTRIBUTING.md
│   ├── PLATFORM_SUPPORT.md
│   ├── THREAT_MODEL.md
│   └── ROADMAP.md
└── .github/
    ├── workflows/
    ├── ISSUE_TEMPLATE/
    └── PULL_REQUEST_TEMPLATE.md
```

## Engineering and open-source requirements

The implementation must include:
- `README.md`, `LICENSE` (MIT, subject to dependency/asset license review), `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, and `CHANGELOG.md`.
- `.github/workflows/` CI for formatting, linting/static analysis, tests, and builds on Linux, macOS, and Windows where supported, plus issue and pull request templates.
- `docs/ARCHITECTURE.md`, `docs/THREAT_MODEL.md`, and `docs/ROADMAP.md`.
- A `Makefile` or equivalent cross-platform task runner, `.editorconfig`, `.gitignore`, `.env.example`, tests, sanitized fixtures, centralized app/repository metadata, and documented release packaging for Linux/macOS/Windows.
- No comments in source code, no hardcoded secrets, no filler code, and no claims of passing tests unless the tests were run.

## Full prompt for Arena.ai/Agent

Copy the entire prompt below into Arena.ai/Agent:

---

# Argus ASM — Attack Surface Management Platform

**Suggested GitHub repository:** `argus-asm`

**GitHub description:** A defensive platform for authorized asset inventory, domain/subdomain tracking, service exposure records, vulnerability imports, risk prioritization, change history, and remediation workflows.

## Languages and technology decisions

- **Primary language:** Go for the core domain, concurrent network work, ingestion, and performance-sensitive processing. Choose Rust instead when memory safety or parsing throughput makes it a better fit. Do not use Python as the core application language.
- **User interface:** Go CLI/TUI + optional Fyne desktop GUI. Build a local command-line application with an optional terminal UI; do not build a website, browser frontend, SaaS, or hosted dashboard.
- **Persistence:** SQLite.
- **Language rationale:** Go owns API, asset inventory, scheduling, integrations, and risk calculations; the interface is a local terminal UI. Use Python only for optional offline data-analysis utilities, not the main service.
- Prefer compiled, statically typed, performance-conscious code for the core. Do not make Python the primary application language. Python may be used only for a clearly justified auxiliary script or offline experiment; it must not be required to run the main product.
- Choose mature, maintained libraries rather than writing cryptography, protocol implementations, parsers, or database abstractions from scratch.
- Pin or bound dependencies, explain major choices, and avoid adding frameworks or dependencies without a concrete need.

## Your role

Act as a principal software engineer, systems architect, cybersecurity specialist, network engineer, performance engineer, QA lead, and open-source maintainer. Work with the discipline expected of a production engineering team. Be persistent and confident, but prove progress with working code, measured behavior, reproducible tests, and accurate documentation. The goal is a high-quality open-source project, not an impressive-looking scaffold.

## Product mission

Build an enterprise-style ASM platform with authorized asset scopes, verified ownership records, domain/subdomain/IP/certificate/service inventory, passive discovery where practical, allowlisted active checks, importers for scanner JSON/CSV, deduplication, asset relationships, explainable risk scoring, findings lifecycle, remediation tickets, change detection, notifications, RBAC, search, exports, and a safe local lab. Never scan arbitrary third-party targets by default.

## Mandatory product constraints: no website and €0 operating cost

- Do not build a website, web application, browser frontend, SaaS, or hosted dashboard. The deliverable is a local CLI tool, optionally with a terminal UI (TUI) or native desktop UI only if it is justified and can be built with free tools.
- All essential functionality must work locally for €0. Do not require paid APIs, paid cloud services, subscriptions, commercial databases, paid scanners, paid feeds, or paid developer tools.
- Use free/open-source dependencies and local storage (SQLite/files) by default. Optional integrations must have an offline alternative using documented fixtures; never make credentials mandatory for the basic workflow.
- Do not make the user deploy or host anything. No required Docker image registry, domain, VPS, cloud account, or externally hosted service.
- If a local HTTP endpoint is genuinely necessary for an integration, it must be optional, disabled by default, and bind to 127.0.0.1 by default. Prefer CLI/TUI interaction over HTTP.
- Document a zero-cost setup, build, test, and run path. Be explicit about optional integrations and any limits of free tiers.

## Mandatory cross-platform, professional UX, and zero-cost constraints

1. Support **Linux, macOS, and Windows**. Provide exact build, test, package, and run commands for each OS. Add CI jobs for all three platforms where technically supported, and report platform gaps honestly.
2. Use **Go** as the default primary language for network/security tooling and concurrent collectors. Use **Rust** when memory safety, parsing throughput, or performance makes it the better fit. Python must not be the core application language. Do not add multiple languages without a clear architectural reason.
3. Do not build a website, browser frontend, SaaS, hosted dashboard, or public web server. Deliver a local-first application. Use a professional CLI/TUI for every tool and a cross-platform **native desktop GUI** where it materially improves usability or visualization. Use Fyne for Go or egui/Slint for Rust when suitable; do not use a browser-based UI.
4. The UI must look intentionally designed and production-ready: consistent typography and spacing, a restrained palette, clear hierarchy, useful navigation, keyboard support, accessible contrast, real loading/empty/error/success states, useful filtering/search, and coherent interactions. Avoid generic AI dashboard templates, gratuitous gradients, fake charts/metrics, lorem ipsum, placeholder buttons, and visual clutter.
5. Every visible control must perform a real action. Demo data must be clearly labeled. All core workflows must work offline with local files/SQLite and included fixtures. Do not require paid API keys, paid services, subscriptions, cloud accounts, hosting, domains, or a hosted database.
6. Performance must be demonstrated with reproducible benchmarks and realistic workloads, not unsupported speed claims. Use bounded concurrency, cancellation, timeouts, streaming for large inputs, and profiling where appropriate.
7. Use portable filesystem, process, signal, permission, and networking behavior. Add tests for Windows path semantics and platform-specific code. Do not claim OS support until the relevant build/test workflow passes.
8. No source-code comments, no hardcoded secrets, no fabricated capabilities, no fake test results, and no artificial LOC padding. Prefer fewer complete features with high reliability over a large number of unfinished features.
9. Include an accessible local demo mode, deterministic fixtures, dependency and license review, release packaging instructions for Linux/macOS/Windows, and clear troubleshooting docs.
10. The core workflow must cost **€0**. Optional external integrations must be clearly marked optional and must have a usable offline alternative.

## Mandatory application identity, About, source, and reporting features

1. If the application has an icon, create a **custom original icon** for this project. Supply editable SVG source and platform assets as applicable: `.ico` for Windows, `.icns` for macOS, and SVG/PNG plus desktop-entry integration for Linux. No generic placeholder icon or third-party trademark.
2. Every GUI application must include an **About** screen/dialog accessible from the main menu or settings. Show app name, icon, version, purpose, license, maintainer/copyright only when known, build/runtime details where useful, and dependency/license notices.
3. Add a **Source Code** action in About that opens the actual GitHub repository URL configured for the project. Add a separate **“Found a bug? Please report it on GitHub.”** action that opens the repository issue tracker/new-issue page when configured. Store URLs centrally. Never invent or hardcode a fake URL; if the repository is not configured, show an honest fallback and disable the external action.
4. For CLI-only tools, provide an `about` command displaying name, version, license, repository URL if configured, OS/architecture, and build information. Provide `--help`, `--version`, stable exit codes, and machine-readable JSON output where useful.
5. Include a safe **Copy diagnostic report** feature for GUI apps and a `diagnostics` command for CLI tools. Preview before saving/copying. Include app version, OS/architecture, runtime/build details and sanitized settings status only. Never include secrets, tokens, personal file contents, or sensitive network data by default.
6. Add contextual help, searchable help where useful, keyboard shortcuts, copy/export, safe settings defaults, first-run guidance, actionable errors with next steps, cancellation, and confirmation for destructive actions.
7. No telemetry or analytics by default. Disclose external network requests, redact secrets from logs, use OS-appropriate config directories and restrictive file permissions, and provide dry-run mode for risky operations.
8. Every UI action must have real behavior and tests. No decorative controls, fake metrics, placeholder links, fabricated test results, or UI shell disconnected from functionality.
9. Add version/changelog and platform-specific install, upgrade, uninstall, reset, and troubleshooting instructions. Provide checksums and signed artifacts where feasible using free tools.
10. Keep all existing constraints: Linux/macOS/Windows support, high-performance Go/Rust core, no website/browser frontend, no required paid services, €0 core operation, no source-code comments, MIT license, SECURITY.md, CONTRIBUTING.md, tests, and professional visual design.

## Mandatory professional engineering workflow: correctness first

1. Do not attempt to generate the entire product in one pass. First write requirements, acceptance criteria, threat model, architecture decisions, and a prioritized implementation plan. Then implement small, complete, testable increments.
2. Before each feature is considered complete, ensure it compiles, passes its tests, handles failure cases, is documented, and works end-to-end. No TODO-only implementation, fake functionality, empty handlers, stubbed return values, or buttons that only look functional.
3. Maintain a requirements-to-tests checklist. Every requirement must map to an implementation and a test, or be explicitly marked out of scope. Add regression tests for every discovered bug.
4. Treat documentation as a release-blocking deliverable, not a final optional pass. Create the complete `docs/` set listed below early in implementation, then update it alongside each feature. Do not leave empty files, generic filler, TODOs, imagined commands, or undocumented configuration.
5. Before reporting completion, execute real local build, test, lint/format, static-analysis, and applicable security/dependency/secret checks. Capture exact command lines and exit statuses. If a tool is unavailable or a platform cannot be tested, say so plainly and distinguish verified results from unverified claims.
6. Validate documentation against the implementation: run every documented command that can safely be run in CI, check that flags and configuration names exist, verify internal links and examples, and ensure root README, changelog, CLI help, package metadata, and docs agree. Add a docs/link/example consistency check to CI where practical.
4. Run and fix the full relevant quality suite: formatter, compiler/build, linter/static analysis, unit tests, integration tests, end-to-end tests where appropriate, dependency audit, secret scan, and platform builds. Do not suppress warnings/errors just to make CI pass.
5. Test valid, invalid, empty, boundary, malformed, oversized, interrupted, timed-out, offline, permission-denied, and partial-failure scenarios. Use deterministic fixtures. Add fuzz/property tests for parsers and input-heavy code; use race/concurrency checks where supported.
6. Validate all untrusted inputs and paths. Use timeouts, cancellation, bounded concurrency, explicit resource limits, safe defaults, least privilege, redacted logs, and actionable errors. Do not silently swallow errors or expose secrets in output.
7. Support Linux, macOS, and Windows with actual CI jobs where possible. Test OS-specific path handling, process management, permissions, signals, networking and packaging. State clearly when a platform or check has not been tested.
8. Benchmark only after correctness is established. Include reproducible benchmarks and describe the test workload and environment. Never invent speed claims, coverage percentages, test results, or security scan results.
9. Before release, require passing builds, tests, lint/static analysis and security checks; no unresolved critical/high-severity defects unless explicitly documented with an accepted risk and mitigation. Include release notes, known issues, rollback/recovery guidance, and checksums where feasible.
10. At the end of the work, provide a truthful quality report listing the exact commands run, platforms tested, pass/fail counts, skipped checks, known limitations, and remaining risks. Do not claim “bug-free” or “fully tested” without evidence.
11. Keep all product constraints: high-performance Go/Rust core, professional functional UI, custom icon and About/source/bug-report links, zero-cost local-first core, no website, Linux/macOS/Windows support, no source-code comments, MIT license, SECURITY.md, CONTRIBUTING.md, and well-structured documentation.

## Required codebase structure

Use the following as the starting point and adapt it to the application’s real needs. Every included directory must have a real purpose; do not create empty scaffolding.

```text
argus-asm/
├── README.md
├── LICENSE
├── CONTRIBUTING.md
├── SECURITY.md
├── CODE_OF_CONDUCT.md
├── CHANGELOG.md
├── .editorconfig
├── .gitignore
├── .env.example
├── Makefile
├── cmd/
│   └── argus-asm/
├── internal/
│   ├── config/
│   ├── domain/
│   ├── service/
│   ├── repository/
│   ├── transport/
│   └── observability/
├── api/
├── tui/
│   ├── views/
│   ├── components/
│   ├── formatting/
│   └── tests/
├── migrations/
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
├── testdata/
├── scripts/
├── docs/
│   ├── README.md
│   ├── INSTALLATION.md
│   ├── USAGE.md
│   ├── CONFIGURATION.md
│   ├── COMMANDS.md
│   ├── ARCHITECTURE.md
│   ├── TROUBLESHOOTING.md
│   ├── SECURITY.md
│   ├── TESTING.md
│   ├── CONTRIBUTING.md
│   ├── PLATFORM_SUPPORT.md
│   ├── THREAT_MODEL.md
│   └── ROADMAP.md
└── .github/
    ├── workflows/
    ├── ISSUE_TEMPLATE/
    └── PULL_REQUEST_TEMPLATE.md
```

### Structural rules
- `cmd/` contains executable entry points only.
- `internal/domain/` contains core models and business rules independent of transport/UI.
- `internal/service/` contains use cases and orchestration.
- `internal/repository/` contains persistence and external data adapters behind clear interfaces.
- `internal/transport/` contains HTTP, CLI, or protocol transport boundaries.
- `tui/` contains the terminal UI and must not duplicate authoritative business rules.
- `api/` contains OpenAPI or protocol definitions where applicable.
- `migrations/` contains versioned database migrations.
- `tests/` contains integration and end-to-end tests; keep unit tests close to their packages if that is idiomatic.
- `testdata/` contains synthetic, sanitized fixtures only.
- `scripts/` contains reproducible developer/CI helpers, not hidden deployment side effects.
- Keep modules cohesive, avoid giant files and circular dependencies, and document deviations from this layout.

## Engineering standards: no vibe coding
- Make evidence-based engineering decisions. Do not invent features, test results, benchmarks, integrations, or security claims.
- Do not generate filler code to inflate line count. Favor complete vertical slices, maintainability, performance, and correctness.
- Inspect the repository first; preserve valid existing work and identify the current state before changing files.
- Write a short plan, then implement and verify the product. Do not stop at a plan, mockup, or architecture diagram.
- Use clear module boundaries, typed interfaces, cohesive packages, explicit error handling, input validation, structured logging, migrations, and reproducible builds.
- Avoid giant files, god objects, circular dependencies, duplicated business logic, unnecessary abstractions, dead code, fake data in production paths, and placeholder buttons.
- Add unit, integration, API, and end-to-end tests appropriate to the system. Test invalid inputs, edge cases, permissions, failure modes, and security boundaries.
- Run formatter, linter, static analysis/type checks, tests, dependency checks, and production build. Fix failures and report only checks actually run.
- No comments in application or test source code: no inline comments, block comments, or commented-out code. Express intent through names, types, tests, and external documentation. Keep required license headers only if legally necessary and place licensing in LICENSE rather than adding routine source comments.
- Never claim the code is bug-free. Provide verified test evidence and list remaining limitations honestly.
- Do not optimize for a target line count. Do not claim 457,000+ LOC unless measured by a reproducible tool and supported by real, non-generated functionality.

## Required repository files
- `README.md`: concise project entry point with purpose, scope, implemented features, supported platforms, prerequisites, quick start, configuration overview, core commands, testing, security notes, limitations, and links into `docs/`. Keep it synchronized with the implementation; never use fabricated screenshots or placeholder claims.
- `LICENSE`: MIT License, only if the author intends to distribute this project under MIT. Verify that all bundled dependencies and copied assets are compatible; do not claim third-party code is MIT.
- `CONTRIBUTING.md`: development setup, branch/commit conventions, issue and PR expectations, testing requirements, code style, security-sensitive changes, and review checklist.
- `SECURITY.md`: supported versions, private vulnerability reporting instructions using a repository-appropriate contact/channel placeholder, response expectations, scope, and responsible disclosure guidance. Do not invent an email address.
- `CODE_OF_CONDUCT.md`: Contributor Covenant or a clearly documented equivalent.
- `CHANGELOG.md`: versioned user-visible changes, starting with an initial unreleased section.
- `docs/` is mandatory in every repository/application and must contain maintained, useful documentation, not empty scaffolding. Required files:
  - `docs/README.md`: documentation index, audience, navigation, and links to the canonical docs.
  - `docs/INSTALLATION.md`: prerequisites and exact clean-install/build steps for Linux, macOS, and Windows, including package/release installation and platform-specific caveats.
  - `docs/USAGE.md`: real end-to-end workflows, examples, expected output, input/output formats, and safe demo-mode usage.
  - `docs/CONFIGURATION.md`: every supported configuration option, type, default, precedence, validation rule, environment variable, file location, security implication, and example.
  - `docs/COMMANDS.md`: every CLI/TUI command, flag, subcommand, exit code, and copy-pasteable example; for GUI-only actions, document equivalent workflows.
  - `docs/ARCHITECTURE.md`: modules, responsibilities, data/control flow, storage, concurrency, trust boundaries, dependency choices, and key design decisions.
  - `docs/TROUBLESHOOTING.md`: common failures, diagnostic steps, recovery, logs, permissions, networking, and platform-specific fixes.
  - `docs/SECURITY.md`: security model, secure defaults, data handling, permissions, network behavior, limitations, and safe operation; must agree with root `SECURITY.md`.
  - `docs/TESTING.md`: test layers, prerequisites, exact commands, fixtures, coverage strategy, lint/static-analysis/security checks, CI matrix, and known gaps.
  - `docs/CONTRIBUTING.md`: local development, code style, branch/commit/PR process, review criteria, and required checks; must agree with root `CONTRIBUTING.md`.
  - `docs/PLATFORM_SUPPORT.md`: verified OS/architecture support, build/run/package instructions, platform-specific behavior, and unverified combinations.
  - `docs/THREAT_MODEL.md`: assets, adversaries, trust boundaries, abuse cases, mitigations, residual risks, and assumptions.
  - `docs/ROADMAP.md`: prioritized milestones with measurable completion criteria.
- Keep the root `README.md` as the concise entry point and link to `docs/`; do not substitute the prompt/specification README for the implementation's operational documentation.
- Documentation must describe implemented behavior only. Mark planned or unsupported behavior explicitly, and keep commands, flags, configuration, screenshots, feature lists, and platform claims synchronized with the actual code and CI. Update docs in the same change as every user-visible feature or configuration change.
- `.github/`: CI workflows for formatting/linting, tests, static checks, build, and dependency/security scanning where supported; issue and pull request templates.
- `.editorconfig`, `.gitignore`, and `.env.example` with placeholders only.
- Tests, fixtures, migrations, and scripts organized in predictable directories.

## Required architecture and repository layout
Use this as the default shape, adapting it to the chosen language and project. Do not create empty directories merely to match a template.

```text
.
├── README.md
├── LICENSE
├── CONTRIBUTING.md
├── SECURITY.md
├── CODE_OF_CONDUCT.md
├── CHANGELOG.md
├── .editorconfig
├── .gitignore
├── .env.example
├── Makefile
├── cmd/
├── internal/
├── pkg/
├── api/
├── tui/
├── migrations/
├── tests/
├── testdata/
├── scripts/
├── docs/
│   ├── README.md
│   ├── INSTALLATION.md
│   ├── USAGE.md
│   ├── CONFIGURATION.md
│   ├── COMMANDS.md
│   ├── ARCHITECTURE.md
│   ├── TROUBLESHOOTING.md
│   ├── SECURITY.md
│   ├── TESTING.md
│   ├── CONTRIBUTING.md
│   ├── PLATFORM_SUPPORT.md
│   ├── THREAT_MODEL.md
│   └── ROADMAP.md
└── .github/
    ├── workflows/
    ├── ISSUE_TEMPLATE/
    └── PULL_REQUEST_TEMPLATE.md
```

Adapt directories where the framework requires another convention; explain important architectural choices in documentation. Keep core business logic independent from HTTP handlers and UI components. Do not expose internal packages as public APIs without a reason.

## Security and open-source release bar
- Secure defaults, least privilege, authentication and authorization where applicable, parameterized database access, safe output encoding, rate limiting, audit logging, and explicit secret redaction.
- No hardcoded credentials, production tokens, personal data, unsafe default scan targets, or external network calls in tests unless explicitly mocked.
- Use pinned or bounded dependencies, generate a dependency inventory/SBOM where practical, review licenses, and document update policy.
- CI must run on a clean checkout. Provide deterministic seed data and a local demo mode.
- Include a `Makefile` or equivalent task runner with documented targets such as `test`, `lint`, `format`, `typecheck`, `build`, and `check`.
- Provide OpenAPI documentation for HTTP APIs or complete CLI help for CLI-first tools.
- Keep UI accessible and responsive; all displayed operational metrics must come from real application data or clearly labeled demo fixtures.
- Use conventional commits or document the selected commit convention. Make pull requests reviewable and changes focused.


## Project-specific acceptance criteria
- Deliver a locally runnable end-to-end MVP using synthetic demo data without paid accounts or production credentials.
- Provide the core domain workflow, persistence where appropriate, API or CLI, and a usable responsive UI when a UI is relevant.
- Implement actual behavior behind every visible control; no fake buttons or hardcoded production metrics.
- Include migrations, seed data, health/readiness checks, structured logs, graceful shutdown, and sensible timeouts where appropriate.
- Add API contract tests and document API authentication, authorization, and rate limits.
- Include a `.env.example` with placeholders only and fail safely when required configuration is missing.
- Include the required open-source governance and security files listed above. The MIT license must be a valid standard license text and must not misrepresent third-party dependencies or assets.
- Use GitHub Actions CI to run formatting, linting/static checks, tests, build, and dependency/license checks where practical. Use least-privilege workflow permissions.
- Document how to run each verification command locally and make CI reproducible.
- Include a short architecture decision record for significant choices such as storage, queueing, parsing, or provider abstractions.

## Execution protocol
1. Inspect the repository and report its state in your working notes; do not overwrite useful work.
2. Create a prioritized plan and implement the highest-value vertical slice immediately.
3. Compile/build early and often. Run tests after each meaningful subsystem; do not defer all validation to the end.
4. Perform a security review of trust boundaries, auth, validation, secret handling, dependencies, and dangerous defaults.
5. Review the complete diff. Search for TODO/FIXME, placeholders, commented-out code, leaked secrets, duplicated logic, and unsupported claims.
6. Run formatting, lint/static analysis, tests, type checks if relevant, production build, and dependency/license checks. Fix discovered issues and rerun.
7. Update README and docs to match the implementation, not the intended future state.
8. Finish with an honest report of completed functionality, exact commands and outcomes, known limitations, and prioritized next milestones.

## Final rule

Do not settle for vibe-coded output. Do not generate filler to inflate LOC, do not pretend incomplete features are finished, do not guarantee zero bugs, and do not report checks that were not actually run. If the full product is too large for one execution, deliver a correct and testable foundation first and then proceed through the roadmap in ranked milestones. Keep all active and test source code free of comments, as requested.


---

## Release checklist

- [ ] Main workflow works end to end with local demo data.
- [ ] Clean-checkout setup instructions have been tested.
- [ ] Formatting, lint/static checks, tests, and production build pass.
- [ ] CI workflows run with least privilege.
- [ ] `SECURITY.md` provides a real maintainer-controlled reporting path before public release.
- [ ] MIT licensing and third-party license compatibility have been reviewed.
- [ ] Documentation describes implemented behavior and known limitations accurately.
- [ ] No secrets, private data, commented-out code, or unexplained stubs are present.
