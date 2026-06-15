---
name: appkit-code-review
description: Use when reviewing macOS AppKit Swift code before committing — checks MVVM separation against the headless Core, Swift 6 concurrency + @MainActor correctness, retain cycles, accessibility identifiers, semantic colors/typography, security (Keychain, App Sandbox, security-scoped bookmarks), view recycling, and localization, catching what the build and UI tests miss.
---

# AppKit Code Review

Run this **after the app builds, before you commit** the `macOS/Sources/App` surface. It catches what isn't a build error and isn't visible in UI tests — code that compiles and runs but is wrong, fragile, or slow. For _writing_ the code, see `appkit-setup` and `appkit-design`; for the spec workflow, see `implementing-a-spec`; for closing out, `verification-before-completion`.

The provable domain lives in the headless `Core` SwiftPM package and is covered by `specify verify` (Swift Testing, `.spec(...)`/`.scenario(...)` traits). This review is mostly about the **view layer** — the AppKit code Core's tests can't reach.

## Lean on tooling first, then judgment

1. **Compiler warnings.** Build with `mise run -C macOS build` and read the log. In Swift 6 language mode, **data-race and main-actor isolation violations are diagnostics** — must-fix, not noise.
2. **swift-format.** Lint against the committed `.swift-format` (lineLength 100, 4-space). It catches force-unwrap/force-try and formatting, but **has no semantic AppKit rules** — the checks below need eyes plus `grep`:
   ```bash
   mise run -C macOS lint
   grep -rnE 'DispatchQueue\.main\.sync|\.wait\(\)' macOS/Sources           # main-thread blocking / deadlock
   grep -rnE 'NSColor\((red|calibratedRed|srgbRed|deviceRed):' macOS/Sources # hardcoded RGB (won't theme)
   grep -rnE 'NSFont\(name:' macOS/Sources                                   # hardcoded font (ignores type ramp)
   grep -rnE '\bUI(View|ViewController|Color|Button|Label|TableView)\b' macOS/Sources # UIKit leaked into AppKit
   ```
3. **Verify symbols you flag.** Before asserting a symbol is wrong or an `@available` is off, run `sdk-api check <Type.symbol>` and `sdk-search` for the canonical pattern — review findings must be grounded, not guessed (see `appkit-setup` for the tools).

## Architecture (MVVM against Core)

- View models are the `@Observable @MainActor` types in **`Core`** — they hold no AppKit view types (`NSView`, `NSColor`, `NSImage`, `NSViewController`). State is exposed as plain values (`String`, `Bool`, enums).
- `NSViewController`/`NSWindowController` do wiring, sheet/dialog coordination, and rendering — **never business logic** (that's spec-provable and belongs in Core).
- View controllers carry `// SPEC: manual` (no cross-target contract) or `// SPEC: <vm-id> (deviates: <reason>)`.
- Rendering is driven from an `Observations` async sequence on `@MainActor`, not manual KVO (see `appkit-setup`). Collections update via diffable snapshots, not in-place array swaps on a live data source.

## Swift 6 concurrency & main-actor correctness

- UI types and view-model entry points annotated `@MainActor`.
- No UI mutation off the main actor; hop back via a `@MainActor` method or `await MainActor.run { }`.
- **No main-thread blocking:** no `DispatchQueue.main.sync`, no semaphore `.wait()` on main, no sync `.result` waits — these deadlock the UI.
- Heavy/IO work runs off-actor (`Task.detached` / `await someActor.work()`), results applied on `@MainActor`.
- `Sendable` respected across actor boundaries; no captured non-`Sendable` mutable state in concurrent closures. No data race left unaddressed by the Swift 6 checker.

## Memory management

- No retain cycles: `[weak self]` in escaping closures that outlive the call; `weak` delegates and target references where Cocoa expects them.
- The `NSWindowController` (and any top-level controller) is **retained for its lifetime** — premature dealloc makes the window vanish. The scaffold's `AppDelegate` holds it; new windows need the same.
- The `render` `Task` driving `Observations` is stored and **cancelled in `deinit`**.
- Observers removed (`NotificationCenter`, KVO) on deinit unless using the block/`NSKeyValueObservation` token form. Timers invalidated/cancelled; no strong `target` cycles.

## Accessibility

- `setAccessibilityIdentifier` on **every interactive control** — unique, stable, and **separate from the label** (UI tests in `appkit-ui-testing` depend on it). The scaffold's `todo-count` field shows the shape.
- `setAccessibilityLabel` on icon-only controls / controls with no visible text.
- Semantic controls (`NSButton`, `NSPopUpButton`, …), not click handlers on plain `NSView`s.
- Keyboard-operable end-to-end (`nextKeyView` loop, `keyEquivalent`s); focus ring visible. No meaning conveyed by color alone.

## Theming & typography

- **Semantic `NSColor` only** — `.labelColor`, `.secondaryLabelColor`, `.controlAccentColor`, asset-catalog named colors. Never hardcoded RGB for chrome (it won't follow Dark Mode / accent / increased-contrast).
- **Semantic typography** — `NSFont.preferredFont(forTextStyle:)` (the scaffold uses `.title1`), not `NSFont(name:size:)`.
- Translucency via materials (`NSVisualEffectView`); **Liquid Glass is opt-in** (`NSGlassEffectView`, gated with `sdk-api`-verified `@available`) — not applied by default. See `appkit-design`.
- Window/view sizing is **content-derived** (`fittingSize` / Auto Layout), never a magic frame; resizable windows size to their constraints.
- Spacing on the 4/8 pt scale; system `controlSize`.

## Security

- No secrets/API keys/tokens in source — credentials via **Keychain** (`SecItem*` or a wrapper).
- App Sandbox / hardened-runtime entitlements scoped to least privilege.
- User-selected file access via `NSOpenPanel`/`NSSavePanel` + **security-scoped bookmarks**, not raw absolute paths.
- External input validated; no shelling out with unsanitized input (`Process` args, never a shell string). HTTPS with App Transport Security on; never disable TLS validation. `WKWebView` (if used) hardened. No logging of PII/tokens.

## Performance

- Long/dynamic lists use `NSTableView`/`NSOutlineView`/`NSCollectionView` (view recycling), not a giant stack of subviews; cell views **reused**, not rebuilt per reload.
- Off-main parsing/IO/compute; UI never blocked. Expensive results cached; images downsampled to display size before assignment.
- `layoutSubtreeIfNeeded`/forced layout not called in hot paths; constraints not churned per frame. Scoped cleanup (`defer`, scoped `FileHandle`/streams) for disposables.

## Localization

- User-facing strings via `String(localized:)` / a string catalog (`.xcstrings`) — not hardcoded.
- Dates/numbers via `FormatStyle` with the current locale — not manual formats. No concatenation for messages; use localized format strings with pluralization.
- Layout uses leading/trailing anchors and adapts to RTL.

## Review report

Summarize each finding with **file + line**, a **severity** (Error must-fix / Warning should-fix / Note could-improve), and a **concrete Swift fix**. If the change touches behavior the spec proves, re-run `specify verify` before claiming done (`verification-before-completion`).

First-party references: [AppKit](https://developer.apple.com/documentation/appkit) · [Swift concurrency](https://docs.swift.org/swift-book/documentation/the-swift-programming-language/concurrency/) · [Accessibility (HIG)](https://developer.apple.com/design/human-interface-guidelines/accessibility) · [Keychain](https://developer.apple.com/documentation/security/keychain_services) · [Security-scoped bookmarks](https://developer.apple.com/documentation/foundation/nsurl#1663783).
