---
name: appkit-migration
description: Use when migrating an existing app into this repo's macOS AppKit surface — UIKit/Mac Catalyst → AppKit (type + idiom remap, navigation rework), Electron/web → AppKit (rewrite the UI, reuse the logic), or Objective-C → Swift incrementally behind a bridging header. Complementary to `appkit-design` (the target idioms), `appkit-setup` (scaffold), and `ios-development` (the shared view-model layer).
---

# AppKit Migration

This skill covers **bringing an existing app into the macOS AppKit surface** under `macOS/Sources/App` (`AppDelegate` + window controllers). The destination shape is fixed by the scaffold: the spec-provable domain and `@Observable` view models live in the headless `Core` SwiftPM package (run by `specify verify`); AppKit is only the view edge that consumes them. **A migration converges on that split — keep (or re-create) the view-model layer, diverge only at the view.** For what the AppKit surface should look like once you get there, see `appkit-design`.

## Pick the migration first

Three very different starting points land here. The work is not the same — identify yours before touching code:

- **UIKit / Mac Catalyst → AppKit.** A UIKit codebase becoming a true AppKit app. Mostly a **type-and-idiom remap** (UIKit class → AppKit class) plus reworking navigation, which has no AppKit equivalent. Domain + view-model code is already platform-agnostic — lift it into `Core` mostly unchanged.
- **Electron / web → AppKit.** A **rewrite of the UI**, not a port: the JS/HTML surface is replaced by AppKit; only domain logic and assets carry over. Plan it as a fresh AppKit app (`appkit-setup` + `appkit-design`) whose view models you author in `Core`.
- **Objective-C → Swift (already AppKit).** Same framework, new language. Migrate **file by file** behind the Obj-C / Swift bridging header; AppKit APIs stay the same.

## Ground every symbol before you write it

Migration is where invented symbols and wrong `@available` gates ship — you're translating from a framework you remember, into one you must verify. **Two CLIs, no exceptions** (built from the apple-platform-tools monorepo into `~/.local/bin` via `mise run install` — referenced, never vendored):

```bash
sdk-api check NSSplitViewController.addSplitViewItem   # exists? + min macOS / deprecation
sdk-search "sidebar detail split view" "unified toolbar"  # canonical AppKit pattern, one query per feature
```

Verify the AppKit symbol you're mapping *to*, and `sdk-search` the canonical pattern, before writing the replacement. Never guess a `UI…`→`NS…` equivalence from memory.

---

## UIKit / Mac Catalyst → AppKit

**1 — Audit the UIKit source.** Inventory the UIKit surface before writing anything:

```bash
grep -rEn '\bUI[A-Z][A-Za-z]+' --include='*.swift' . | sort | uniq -c | sort -rn | head -60
```

List the controls, the navigation model (nav controller / tab bar / modal), custom views, touch handling, and any `#if targetEnvironment(macCatalyst)` branches (they mark where AppKit behavior was already bolted on — delete the shims once native).

**2 — Lift the logic into `Core`, scaffold the surface.** The `@Observable` view models and domain types are UI-agnostic; move them into the `Core` package as-is and prove them with the Swift Testing `.spec(…)`/`.scenario(…)` traits (see `ios-development`). Scaffold the empty AppKit window with `appkit-setup` and get it building **before** porting any screen. Keep the old project to read from.

**3 — Map the types.** Verify each AppKit symbol with `sdk-api check` as you go:

| UIKit | AppKit | Note |
|---|---|---|
| `UIView` / `UIViewController` | `NSView` / `NSViewController` | `NSView` origin is **bottom-left** — override `isFlipped` for top-left layout |
| `UIWindow` | `NSWindow` + `NSWindowController` | heavier on macOS; always pair with a controller |
| `UILabel` | `NSTextField(labelWithString:)` | non-editable, borderless, no background |
| `UIButton` / `UISwitch` / `UISlider` | `NSButton` / `NSSwitch` / `NSSlider` | |
| `UITextField` / `UITextView` | `NSTextField` / `NSTextView` in an `NSScrollView` | text view doesn't scroll itself; field commits on end-editing |
| `UISegmentedControl` / `UIStepper` | `NSSegmentedControl` / `NSStepper` | |
| `UIActivityIndicatorView` / `UIProgressView` | `NSProgressIndicator` (`style` switches bar/spinner) | |
| `UITableView` | **view-based** `NSTableView` + `NSTableViewDiffableDataSource` | never cell-based; in an `NSScrollView` |
| `UICollectionView` | `NSCollectionView` + compositional layout | |
| (tree / outline) | `NSOutlineView` | native on macOS, no UIKit twin |
| `UINavigationController` | `NSSplitViewController` **or** content-view replacement + `NSToolbar` | **no drill-down nav** — see step 4 |
| `UITabBarController` | `NSTabViewController` or a source-list sidebar | |
| modal `present(_:)` | a **sheet** (`beginSheet`) or a separate window / `NSPanel` | |
| `UIAlertController` | `NSAlert` as an async sheet (`beginSheetModal(for:)`) | never `runModal()` |
| `UIMenu` / context menu | `NSMenu` | |
| `UIColor` / `UIFont` / `UIImage` | `NSColor` / `NSFont` / `NSImage` | semantic only — see non-negotiables |
| Auto Layout (anchors, `NSLayoutConstraint`) | **same** | constraints port directly — the one thing that stays |
| `UIStackView` | `NSStackView` | |
| `UIGestureRecognizer` / `touchesBegan` | `NSGestureRecognizer` / `mouseDown`/`mouseDragged` | mouse, not touch; prefer target-action / gesture over overrides |
| `traitCollection` (dark mode) | `effectiveAppearance` | observe via `viewDidChangeEffectiveAppearance()` |

**4 — Rework navigation (the real work).** UIKit drill-down has **no AppKit equivalent** — don't emulate a nav stack. Re-express the structure: list→detail becomes an `NSSplitViewController`; a flow of full-screen steps becomes toolbar/segmented-driven content swapping or a sequence of sheets; tab structure becomes a source-list sidebar or `NSTabViewController`. See `appkit-design` Step 1 for the app-type → window-spine mapping.

**5 — Rewire the view edge to `@Observable`.** AppKit has no `updateProperties()`; drive rendering from an `Observations` async sequence consumed on the main actor (pattern in `ios-development`). The view model is the same one you lifted into `Core` — the controller just renders it. Mark migrated controllers `// SPEC: manual` when no cross-target contract applies.

**6 — Fix coordinate + concurrency differences.** Set `isFlipped` on containers laid out top-left. Replace `DispatchQueue.main.async` UI hops with `@MainActor` methods. There's no `safeAreaInsets` the same way — use `contentLayoutGuide` and toolbar/sidebar safe areas (esp. for edge-to-edge Liquid Glass).

**Don't:** recreate push/pop navigation · assume top-left origin · keep `#if targetEnvironment(macCatalyst)` shims (you're native now — delete them). **Do:** lean on the constraints that carried over · port screen-by-screen, building after each.

---

## Electron / web → AppKit

A **rewrite of the UI** — a new AppKit app that reuses logic, never a line-by-line port.

1. **Separate logic from UI.** Reimplement portable JS domain logic as `Core` view models + domain types (proven by `specify verify`). A substantial engine can stay a local service/CLI the app talks to, or be ported incrementally.
2. **Rebuild the UI in AppKit** via `appkit-design` (real controls, HIG, Liquid Glass). **Do not** wrap the web app in a `WKWebView` and call it native — that keeps every Electron downside.
3. **Map web idioms:** HTML form controls → `NSTextField`/`NSPopUpButton`/`NSButton`/`NSSwitch` · flexbox/grid → Auto Layout (`NSStackView`/`NSGridView`) · SPA routing → `NSSplitViewController`/`NSTabViewController`/view swapping · `BrowserWindow` → `NSWindow`+`NSWindowController` · Electron `Menu`/tray → `NSMenu`/`NSStatusItem` · `ipc*` → direct Swift calls / a service layer · `fs` → `FileManager` + `NSOpenPanel`/`NSSavePanel` + security-scoped bookmarks · notifications → `UNUserNotificationCenter` · `localStorage` → `UserDefaults` or Application Support.
4. **Reuse assets** but regenerate the **app icon** at macOS sizes and SF Symbols for toolbar/menu glyphs (`NSImage(systemSymbolName:accessibilityDescription:)`).

**Don't** ship a `WKWebView` shell as "native." **Do** keep portable domain logic; rewrite only the presentation layer.

---

## Objective-C → Swift (already AppKit)

Same framework, new language — migrate **incrementally** behind the bridging header.

1. **Set up bridging.** A mixed target uses an Obj-C bridging header (Swift sees Obj-C) and the generated `-Swift.h` (Obj-C sees Swift). In Tuist, set `SWIFT_OBJC_BRIDGING_HEADER` in the target's `settings:` in `Project.swift`, then `mise run -C macOS generate`. New Swift files can call existing Obj-C immediately.
2. **Migrate leaf files first.** Convert classes with the fewest dependents (models, utilities) one at a time; build after each. Keep the Obj-C version until the Swift one compiles and its `Core` tests pass, then delete it.
3. **Translate idioms:** `@property (strong)`→`var` · `@property (weak) delegate`→`weak var` · `_Nullable`→optionals (**audit each** for true nullability) · `NSArray`/`NSDictionary`→typed `Array`/`Dictionary` · `id`→a concrete type or protocol (`Any` last resort) · `respondsToSelector:`→optional protocol reqs / `?.` · `dispatch_async(main)`→`@MainActor` · KVO→`NSKeyValueObservation` · `@selector`→`#selector` · `#define`→`let`/`enum` · `NSError**`→`throws`.
4. **Keep `@objc`** on anything reached by selector, KVO, Cocoa bindings, or `NSToolbar`/`NSMenu` items — Swift lets you omit it, Cocoa still needs it.
5. **Adopt Swift 6 concurrency last**, once files are Swift — annotate UI types `@MainActor` and resolve isolation diagnostics (see `appkit-code-review`).

**Do:** migrate file-by-file, build/test after each · audit `_Nullable`/`id` rather than blindly making everything non-optional or `Any` · keep `@objc` where Cocoa reaches by selector/KVO.

---

## The non-negotiables apply to the migrated surface

The new AppKit code is held to the same bar as fresh code — these are exactly what a remap skips:

- **Accessibility identifier on every interactive control** (`setAccessibilityIdentifier(_:)`) — a stable, non-localized handle, **separate** from the VoiceOver label (`setAccessibilityLabel(_:)`); both are needed. `appkit-ui-testing` queries the identifier.
- **Semantic colors only** — `.labelColor`, `.secondaryLabelColor`, `.controlAccentColor`, `.textBackgroundColor`. Never carry over a hardcoded RGB/hex from the source.
- **Semantic typography** — `NSFont.preferredFont(forTextStyle:)`, never `systemFont(ofSize:)`.
- **Content-derived window sizing** — `fittingSize` / Auto Layout, never a magic frame ported from the UIKit/CSS layout.
- **Explicit Liquid Glass adoption** — gated `if #available(macOS 26, *)`; a table `style` or sidebar is *not* glass.
- **Swift 6 strict concurrency** — `@MainActor` on UI types; **no force-unwrap, no force-try** (audit every `_Nullable`→`!` temptation).

See `appkit-design` for the full treatment.

## Validate the migration

```bash
# UIKit→AppKit: confirm nothing UIKit survived
grep -rEn '\bimport UIKit\b|\bUI(View|ViewController|Color|Button|Label|TableView)\b' --include='*.swift' macOS/

mise run -C macOS generate      # regenerate after Project.swift / source changes (alias: g)
mise run -C macOS build         # xcodebuild the AppKit app via Tuist (alias: b)
mise run -C macOS launch:macos  # build + launch to eyeball the migrated surface
mise run -C macOS test:app      # app-tier wiring tests (Swift Testing); .xcodeproj/Derived are gitignored
mise run -C macOS fmt           # swift-format, the committed .swift-format (lineLength 100, 4-space)
```

The lifted/rewritten view models and domain run in `Core`: `mise run -C Core test` and `specify verify` (Swift Testing, `swift` report format, `.spec(…)`/`.scenario(…)` traits from `TestSupport`'s `SpecTraits.swift` — the scenario id lives in the trait, never the test name). Then run `appkit-ui-testing` to confirm behavior and `appkit-code-review` for concurrency, memory, accessibility, and theming. Migrated layouts often need resizing — re-derive the window size per `appkit-design` Step 4.

## When to step out of this skill

- Designing the AppKit surface you're migrating to → `appkit-design`
- Scaffolding the app / a Tuist module → `appkit-setup`
- The build/run inner loop and build-error triage → `appkit-dev-workflow`
- Concurrency / memory / a11y / theming review of the new code → `appkit-code-review`
- UI tests against the new identifiers → `appkit-ui-testing`
- Shared `@Observable` / `Observations` / Swift Testing idioms → `ios-development`
- About to write tests, or claim work is done → `test-driven-development`, `verification-before-completion`
- Something behaves unexpectedly after a rebuild → `systematic-debugging`
- Implementing a spec end-to-end → `implementing-a-spec`

## HIG and first-party docs

When a design choice is unclear, read the HIG and `sdk-search` the pattern — don't vendor HIG files.

- [Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines) · [AppKit reference](https://developer.apple.com/documentation/appkit) · [Tuist](https://docs.tuist.dev) · [Swift Testing](https://developer.apple.com/documentation/testing)

The Xcode MCP bridge (build/run/inspect from the agent) is per-machine — it lives in your user/local config, not the committed `.mcp.json`. See `ios-development` for the setup if you use it.
