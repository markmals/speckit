---
name: appkit-dev-workflow
description: Use when building, running, or fixing build errors in the macOS AppKit app surface — the generate/build/launch inner loop via the scaffold's mise tasks, reading stdout + crash logs, and diagnosing build failures. Distinct from the headless Core test loop and from `ios-simulator-control`.
---

# AppKit Dev Workflow

The build/run/launch inner loop for the **macOS AppKit app surface** (`macOS/Sources/App` — `AppDelegate` + window controllers). This is the surface you _run_ to see UI; it is **not** where behavior is proven. The spec-provable domain + `@Observable` view models live in the headless SwiftPM `Core` package, run by `specify verify` (and `mise run -C <dir> test`) with no Tuist, Xcode, or signing. Keep the two loops separate:

- **Proving behavior** → run the Core suite. See `ios-development` and `test-driven-development`.
- **Seeing the app** → this skill.
- **Driving the iOS simulator** → `ios-simulator-control` (a different platform entirely).

Tuist owns the app: `Project.swift` is the source of truth, and the generated `.xcodeproj` / `.xcworkspace` / `Derived` are gitignored. You regenerate them on demand — never hand-edit or commit them.

## The inner loop

All commands go through the scaffold's mise tasks (Tuist is pinned in `mise.toml`). Run them with `-C <dir>` pointed at the app directory:

```bash
mise run -C macOS generate      # tuist generate --no-open (alias: g)
mise run -C macOS build         # generate, then xcodebuild the app (alias: b)
mise run -C macOS launch:macos  # generate, build Debug into Derived/, then `open` the .app
```

`launch:macos` builds and hands off to `open`, which **returns immediately and detaches** — so you won't see the app's `stdout` in your terminal. For a tight change → run → observe loop:

1. Edit `macOS/Sources/App/…`.
2. `mise run -C macOS launch:macos`.
3. Read output (below), or screenshot/inspect via the Xcode MCP.
4. Repeat.

If you run `launch:macos` three times in a row chasing the same thing, you're probably proving behavior — move it down into a Core view-model test instead.

## Reading output: stdout + crashes

`open` detaches the app, so reach for the unified log or run the inner binary directly.

```bash
# Stream the app's stdout/stderr + os_log while it runs (background it so it doesn't block your turn)
log stream --level debug --predicate 'process == "<AppName>"'

# Or run the built binary in the foreground to see stdout directly (no `open`):
Derived/Build/Products/Debug/<AppName>.app/Contents/MacOS/<AppName>
```

Run either form via the Bash tool with `run_in_background: true`, then read the streamed output as you interact.

**Crash on launch** — read the latest crash report; the exception type + backtrace is at the top:

```bash
ls -t ~/Library/Logs/DiagnosticReports/<AppName>-*.ips | head -1 | xargs cat
```

The crash you'll hit most: a window that flashes and the app quits. That's almost always the `MainWindowController` reference not being retained on the `AppDelegate` — fix the lifetime, never paper over it with `LSUIElement` / agent activation. The scaffold's `AppDelegate` already retains its controller; keep that invariant.

## Diagnosing build errors

`mise run -C macOS build` runs `xcodebuild` through Tuist. Swift emits **the full set of errors in one pass** — read them all, batch-fix, then re-run. Don't fix one and rebuild.

| Symptom | Fix |
|---|---|
| `tuist: command not found` / Xcode not selected / license unaccepted | A prerequisite is missing — see below; do **not** self-install. |
| `Cannot find 'X' in scope` / `no such module` | Add the `import`, or add the SwiftPM dependency in `Project.swift` and `mise run -C macOS generate`. |
| `'NSGlassEffectView' is only available in macOS 26 or newer` | Gate with `if #available(macOS 26, *)`, or raise `deploymentTargets` in `Project.swift`. Verify the version with `sdk-api` (below). |
| `Call to main actor-isolated … in a synchronous nonisolated context` | Mark the UI type/method `@MainActor` (all AppKit view/controller types are). |
| Window opens then app quits | Retain the window controller on the `AppDelegate`. |
| Blank/black window | Content view not added, or zero-size constraints — check `contentView` + Auto Layout, not a magic frame. |
| Stale build after a `Project.swift` edit | Regenerate: `mise run -C macOS generate`. |

After fixing, format before you commit: `mise run -C macOS fmt` and `mise run -C macOS lint` (swift-format, the committed `.swift-format` — lineLength 100, 4-space).

## Prefer the Xcode MCP when connected

When Xcode's external-agent MCP is connected, prefer it over shelling out — you get structured build results, test runs, and diagnostics instead of scraping `xcodebuild` output. It is **per-machine** (it needs Xcode running with the feature enabled), so it lives in your **user/local** MCP config, never the committed `.mcp.json`. `ios-development` covers the setup; don't duplicate it here. Fall back to the mise tasks above when the bridge isn't available.

## App-level tests are a secondary surface

`MainWindowController` wiring and other AppKit-only scenarios that can't be proven headlessly run through xcodebuild:

```bash
mise run -C macOS test:app      # tuist test <AppName>
```

These are **not** run by `specify verify` — the spec-provable scenarios live in Core. App-level tests still use Swift Testing with the `.spec(...)` / `.scenario(...)` traits from the shared `TestSupport` target's `SpecTraits.swift`; the scenario id lives in the trait, never the test name. Keep app-tier tests thin — assert wiring (a window builds, a control exists, the accessibility identifier is set), not domain logic.

## Ground every symbol before you write AppKit

Never guess an AppKit symbol name or `@available` version. Before writing app-surface code, verify with the external `sdk-api` / `sdk-search` CLIs (built from the apple-platform-tools monorepo into `~/.local/bin` via `mise run install` — referenced, never vendored):

```bash
sdk-api check NSGlassEffectView          # does the symbol exist, and what's its macOS availability?
sdk-search "toolbar with accessory view" # find the canonical AppKit pattern
```

`sdk-search` is also how you confirm a HIG-driven affordance against canonical patterns; for the guidelines themselves read [Apple's HIG](https://developer.apple.com/design/human-interface-guidelines). For the AppKit framework reference use [developer.apple.com/documentation/appkit](https://developer.apple.com/documentation/appkit). See `appkit-design` for layout, controls, sizing, and Liquid Glass; `appkit-setup` for first-run tooling.

When you do write app-surface code, the non-negotiables still apply: an accessibility identifier on every interactive control (separate from its label); semantic colors only (`.labelColor`, `.controlAccentColor`); semantic typography (`NSFont.preferredFont(forTextStyle:)`); content-derived window sizing (`fittingSize` / Auto Layout, never magic frames); explicit Liquid Glass adoption; Swift 6 strict concurrency with `@MainActor` on UI types; no force-unwrap / force-try.

## Prerequisites

If `tuist` is missing, Xcode isn't selected, or the license is unaccepted: **stop, do not self-install**, and ask the user to run `appkit-setup`, then retry the failed command.

## When to step out of this skill

- Unexpected runtime behavior that survives a rebuild → `systematic-debugging`.
- About to claim the app works → `verification-before-completion` (run the Core suite + actually launch and look).
- Implementing a spec end-to-end → `implementing-a-spec`.

## Commit

`Project.swift` edits go alone (`chore: add <module> to Tuist project`) — never bundled with feature code. Never commit the generated `.xcodeproj` / `.xcworkspace` / `Derived` (gitignored). Asset additions are their own commit. Use the scoped-commits convention (`specify gate scope`).
