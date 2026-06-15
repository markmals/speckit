---
name: appkit-design
description: Use when designing or building any macOS AppKit UI in this repo's app surface — picking the canonical control/layout for a window, adopting Liquid Glass, sizing a window to its content, applying semantic colors and typography, setting accessibility identifiers, or reviewing AppKit UI for design correctness. Targets modern AppKit (macOS 26/27). Complementary to `appkit-setup` (scaffold), `appkit-ui-testing` (queries the identifiers), and `ios-development` (shared view-model/domain idioms).
---

# AppKit Design

This skill covers **how to design and write the macOS AppKit surface** — the view controllers and windows under `macOS/Sources/App`. The spec-provable domain and `@Observable` view models live in the headless `Core` SwiftPM package (run by `specify verify`); this skill is about the view edge that consumes them. For the shared Swift idioms (`@Observable`, `Observations`, Swift Testing traits), see `ios-development`. For the workflow of implementing a spec, see `implementing-a-spec`.

> **You already know most of the controls. That is the trap.** A capable agent reaches for the right control from memory (`NSSplitViewController`, a view-based `NSTableView`, `NSGridView`) and then *skips everything that actually breaks*: invents a symbol that doesn't exist, asserts a false equivalence ("`.inset` gives you Liquid Glass"), hardcodes a window frame, and ships zero accessibility identifiers. **Memory is exactly where the errors are. Verify everything.**

## The grounding mandate (do this first, every time)

**Before writing any AppKit UI, ground it. Two CLIs, no exceptions.** Both are built from the apple-platform-tools monorepo (`mise run install` → `~/.local/bin`); never vendor them into the repo.

1. **`sdk-search`** for the canonical, HIG-grounded pattern — one focused query per feature you need:
   ```bash
   sdk-search "settings sidebar" "file table" "unified toolbar"   # batch: one query per feature
   ```
2. **`sdk-api`** to verify *every* symbol and its macOS availability before you write it:
   ```bash
   sdk-api check NSGlassEffectView.effectIsInteractive   # exists? + min macOS / deprecation
   ```

**Workflow:** front-load all `sdk-search` calls for the screen → read the best patterns → verify the symbols you'll use with `sdk-api` → *then* write code, adapting the snippets. Do not interleave searching with coding.

> **"This is just a code sketch, I'll answer from memory."** No. That sentence is the #1 failure mode — it is how invented symbols and false claims ship. A sketch that names a wrong API is worse than no sketch. Sketch or production, the grounding is the same two commands. Knowing a symbol exists isn't verifying it — `sdk-api check` costs a second and returns the min-macOS you need for the `@available` gate.

## Non-negotiable design hygiene

Apply to **every** design, no matter how small. These are the things a strong model skips:

| Rule | Why it's here |
|------|---------------|
| **Accessibility identifier on every interactive control** — `control.setAccessibilityIdentifier("saveButton")`. This is **not** the VoiceOver label (`setAccessibilityLabel("Save")`); both are needed, they are different things. | The identifier is the stable, non-localized handle `appkit-ui-testing` queries; the label is localized text VoiceOver speaks. |
| **Semantic colors only** — `.labelColor`, `.secondaryLabelColor`, `.controlAccentColor`, `.textBackgroundColor`, `.separatorColor`. Never literal RGB/hex. | Hardcoded colors break Dark Mode, Increase Contrast, and the user's accent tint. |
| **Semantic typography** — `NSFont.preferredFont(forTextStyle: .body)`, not `NSFont.systemFont(ofSize: 13)`. AppKit's signature adds an optional `options:` (default `[:]`), so the no-`options` call also compiles. | Respects the user's text size; the literal `ofSize:` is the most common typography miss. |
| **Content-derived window sizing** — size from the `contentViewController`'s `fittingSize` / Auto Layout, never a magic `900×640` frame. | Hardcoded frames clip or waste space across displays and text scales. |
| **Explicit Liquid Glass adoption** — `NSGlassEffectView` / `NSVisualEffectView` materials and a real, populated unified toolbar, gated with `if #available`. A table `style` or a sidebar gives vibrancy in places, but `.inset` is **not** Liquid Glass and an empty `NSToolbar` renders nothing. | Baselines assert implicit glass they never actually adopt. |
| **Swift 6 strict concurrency** — AppKit UI types are `@MainActor`; no force-unwrap, no force-try. | The repo builds in Swift 6 language mode; isolation and optional-safety are enforced, not optional. |

## Design workflow

### Step 1 — App type → anchor structure

Identify the app type; it fixes the window's spine. Then `sdk-search` the anchor pattern.

| App type | Anchor structure |
|----------|------------------|
| Settings / config / inspector | `NSSplitViewController` (sidebar + content [+ inspector]) |
| Document / editor | `NSWindowController` + content VC + unified `NSToolbar` (populated, via its delegate) |
| Browser / data-heavy | `NSSplitViewController` + view-based `NSTableView` / `NSOutlineView` |
| Peer workflows / paged panes | `NSTabViewController` (toolbar tabs) |
| Menu-bar utility | `NSStatusItem` menu bar extra |
| Single-purpose utility / form | `NSPanel` / window + `NSGridView` form |

The window controllers live under `macOS/Sources/App`; the `AppDelegate` owns them.

### Step 2 — Requirement → canonical control

Map each requirement to a control, then `sdk-search` the pattern — don't write the plumbing from memory.

| Requirement | Canonical control |
|-------------|-------------------|
| Sidebar / source list | `NSSplitViewItem(sidebarWithViewController:)` — never a bare `NSSplitView` |
| Tabular data, selectable/sortable | **view-based** `NSTableView` + `NSTableViewDiffableDataSource` — never cell-based (`NSCell`, `dataCell`) |
| Hierarchy / tree | `NSOutlineView` |
| Tiles / grid | `NSCollectionView` + compositional layout |
| Toolbar | `NSToolbar` + `NSToolbarDelegate` (don't skip the delegate, don't ship it empty) |
| Form (label↔field rows) | `NSGridView` |
| Stacked controls / tool rows | `NSStackView` |
| Modal decision | `NSAlert` as an async sheet (`beginSheetModal(for:)`) — never `runModal()` on the main thread |
| Transient detail | `NSPopover` (`.transient`) |
| Pick one of few / modes | `NSSegmentedControl` |
| On/off | `NSSwitch` |
| File open/save | `NSOpenPanel` / `NSSavePanel` + `UTType` |

Prefer the standard control with configuration over a custom `NSView` reimplementing it — you keep semantic color, focus ring, and accessibility for free. For clicks, use target-action / gesture recognizers, not an `mouseDown:` override.

### Step 3 — Layout, color, typography

Auto Layout via anchors; `NSStackView` / `NSGridView` for structure; HIG spacing metrics (don't invent paddings). Semantic `NSColor` named for the *role* (label, separator, window), not the one whose light-mode value looks right. Semantic `NSFont.TextStyle` (`.body`, `.headline`, `.title2`, …) via `preferredFont(forTextStyle:options:)`; for monospaced/rounded, derive a descriptor with `fontDescriptor.withDesign(.monospaced)` and `NSFont(descriptor:size: 0)` — handle the `nil`, never force-unwrap.

For layer-backed views, `CGColor` is a static snapshot: re-resolve in `viewDidChangeEffectiveAppearance()` inside `effectiveAppearance.performAsCurrentDrawingAppearance { … }`, and also observe `NSWorkspace.accessibilityDisplayOptionsDidChangeNotification` (Increase Contrast / Reduce Transparency are not appearance changes).

### Step 4 — Size the window to its content

Derive the size; never hardcode a `contentRect`. The `contentViewController`'s Auto Layout is the source of truth:

```swift
window.contentViewController = content
content.view.layoutSubtreeIfNeeded()                  // fittingSize is .zero before a layout pass
let derived = content.view.fittingSize                // smallest size satisfying constraints
let size = NSSize(width: ceil(derived.width), height: ceil(derived.height))  // round UP
window.setContentSize(size)
window.contentMinSize = size                          // content area, not the framed minSize
window.setFrameAutosaveName("EditorWindow")           // user's resize survives relaunch
```

There is **no** `systemLayoutSizeFitting(_:)` in AppKit — that's UIKit; use `fittingSize`. Pin content on both axes or `fittingSize` stays `.zero`. For multi-pane windows, the floor is the sum of each `NSSplitViewItem.minimumThickness`.

### Step 5 — Adopt Liquid Glass (macOS 26/27)

Adoption means gating on `if #available(macOS 26, *)` and reaching for the real symbols — not asserting that a sidebar already gives you the look. Verify each min-macOS with `sdk-api availability` and gate everything below your deployment target.

| Need | Use | Min macOS |
|------|-----|-----------|
| Floating, content-forward glass chrome | `NSGlassEffectView` (assign your view to `contentView`, set `cornerRadius` — never `addSubview`) | 26.0 |
| Window / sidebar / header vibrancy by role | `NSVisualEffectView` + `.material` (`.sidebar`, `.headerView`, `.hudWindow`) | 10.10 |
| Merge adjacent glass shapes | `NSGlassEffectContainerView` | 26.0 |
| Glass-bezeled control | `NSButton.bezelStyle = .glass` | 26.0 |
| Scroll-edge fade | `preferredScrollEdgeEffectStyle` on a titlebar / split-view accessory controller | 26.1 |
| Concentric nested corners | `cornerConfiguration` + `.uniformCorners(radius: .containerConcentric)` (read-only — override the getter, never assign) | 27.0 |

`.inset` is a *table style*; an `NSVisualEffectView` is *vibrancy* (10.10) — neither is adopting `NSGlassEffectView` (26.0). Glass changes the material, not the structure: sidebars still use `NSSplitViewController`.

### Step 6 — Accessibility baseline

Identifier on **every** interactive control (`setAccessibilityIdentifier(_:)`, a code-stable token, never localized) *and* a label on every icon-only control (`setAccessibilityLabel(_:)` / `NSImage(systemSymbolName:accessibilityDescription:)`, localized). Custom `NSView`s need `setAccessibilityElement(true)` + `setAccessibilityRole(_:)` to be visible to assistive tech at all, plus `accessibilityPerformPress()` to make the action triggerable. Respect Increase Contrast, Reduce Motion, and Reduce Transparency via `NSWorkspace.shared.accessibilityDisplayShould*`. Verify a role constant with `sdk-api members NSAccessibility.Role` before guessing it.

## Rationalizations — STOP if you think any of these

| Excuse | Reality |
|--------|---------|
| "It's just a sketch, I'll answer from memory." | The sketch is where wrong symbols enter. Run `sdk-search` + `sdk-api` anyway. |
| "I know `NSGlassEffectView` / this color / this font exists." | Then `sdk-api check` costs a second to prove it *and* its min-macOS. Knowing isn't verifying. |
| "The sidebar / `.inset` style already gives the glass look." | Implicit vibrancy in one place ≠ adopting Liquid Glass. Gate `if #available(macOS 26, *)` and adopt it explicitly. |
| "I added an accessibility label, that covers a11y." | Label (VoiceOver text) ≠ identifier (UI-test handle). Set both. |
| "I'll size the window with a frame that looks about right." | Derive it from `fittingSize`. A guessed frame clips on the next display. |

## Verifying the build

After writing UI, run the app surface through mise — Tuist generates the project (the `.xcodeproj` / `.xcworkspace` / `Derived` are gitignored):

```bash
mise run -C macOS generate          # regenerate the Tuist project after manifest/source changes
mise run -C macOS build             # build the AppKit app
mise run -C macOS launch:macos      # build + launch to eyeball the design
mise run -C macOS test:app          # app-target tests (UI); Core specs run via `specify verify`
mise run -C macOS fmt               # swift-format, the committed .swift-format (lineLength 100, 4-space)
```

View-model and domain specs live in `Core` and run with `mise run -C Core test` / `specify verify` (Swift Testing, `swift` report format, the `.spec(…)`/`.scenario(…)` traits from `TestSupport`'s `SpecTraits.swift` — the scenario id lives in the trait, never the test name). UI here is the view edge: mark controllers `// SPEC: manual` when no cross-target behavioral contract applies.

## When to invoke a more specific skill

- Scaffolding the app or adding a module to `Project.swift`? → `appkit-setup`
- Writing UI tests that query these identifiers? → `appkit-ui-testing`
- Shared `@Observable` / `Observations` / Swift Testing idioms? → `ios-development`
- About to write tests, or claim work is done? → `test-driven-development`, `verification-before-completion`
- Debugging something unexpected? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`

## HIG and first-party docs

When a design choice is unclear, read the relevant HIG section and `sdk-search` the pattern — don't vendor HIG files.

- [Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines) · [Materials](https://developer.apple.com/design/human-interface-guidelines/materials) · [Color](https://developer.apple.com/design/human-interface-guidelines/color) · [Typography](https://developer.apple.com/design/human-interface-guidelines/typography) · [Windows](https://developer.apple.com/design/human-interface-guidelines/windows) · [Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility)
- [AppKit reference](https://developer.apple.com/documentation/appkit) · [Tuist](https://docs.tuist.dev) · [Swift Testing](https://developer.apple.com/documentation/testing)

The Xcode MCP bridge (build/run/inspect from the agent) is per-machine — it lives in your user/local config, not the committed `.mcp.json` and is not projected by SpecKit. See `ios-development` for the setup if you use it.
