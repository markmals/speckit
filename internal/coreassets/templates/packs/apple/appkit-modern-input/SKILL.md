---
name: appkit-modern-input
description: Use when handling pointer, drag, text-selection, or keyboard input in this repo's macOS AppKit surface — replacing a mouseDown:/rightMouseDown: override or a nextEvent(matching:) tracking loop with gesture recognizers and control events, wiring table/outline/collection drag-and-drop, custom text selection, Tab key-view navigation, or an NSStatusItem with a custom view or window. Targets modern AppKit (macOS 26/27). Complementary to `appkit-design` (controls/layout) and `appkit-ui-testing` (queries the identifiers).
---

# Modernizing AppKit Input Handling

This skill covers **how to wire user input** in the macOS AppKit surface under `macOS/Sources/App` — the view controllers and windows that consume the `@Observable` view models from the headless `Core` package. The discipline: **prefer intent over tracking loops.** The modern way to handle mouse events is gesture recognizers and dedicated view-based APIs — not `mouseDown(with:)` overrides or `nextEvent(matching:)` loops. For picking the control itself, see `appkit-design`; for the shared Swift idioms (`@Observable`, `Observations`), see `ios-development`.

> **You already know `mouseDown(with:)`. That is the trap.** A capable agent reaches for the override from memory and reinvents selection, context menus, and dragging by hand — then invents a delegate signature or a wrong `@available` version on the way. The modern API is shorter *and* more correct (it handles Control-click, the keyboard menu key, and accessibility for free). **Ground every symbol before you write it.**

## The grounding mandate (do this first, every time)

**Before writing any input code, ground it. Two CLIs, no exceptions.** Both are built from the apple-platform-tools monorepo (`mise run install` → `~/.local/bin`); never vendor them into the repo.

1. **`sdk-search`** for the canonical pattern — one query per input job:
   ```bash
   sdk-search "table drag and drop" "gesture recognizer" "status item custom view"
   ```
2. **`sdk-api`** to verify *every* symbol and its macOS availability — several of these APIs are new in macOS 27, so the `@available` gate is load-bearing:
   ```bash
   sdk-api check NSStatusItemExpandedInterfaceDelegate   # exists? + min macOS
   sdk-api check NSTextSelectionManager                  # new in 27 — gate it
   ```

Never guess a delegate method signature or an `@available` version from memory.

## Replace each `mouseDown:` job with its dedicated API

`mouseDown(with:)` is usually overridden for one of four jobs. Each has a better, more reliable home — find every override and move it:

| Overriding `mouseDown:` to… | Use instead |
|---|---|
| **Track selection** in a table/collection/outline view | Observe the `selected` property on `NSTableRowView` / `NSCollectionViewItem`, **or** the selection delegate callbacks (`NSTableViewDelegate`, `NSOutlineViewDelegate`). |
| **Show a context menu** | `NSView.defaultMenu` (class — same menu per instance), `NSResponder.menu` (per-responder), or `NSView.menu(for:)` (built dynamically from the event). All three also handle Control-click, the keyboard menu key, and accessibility. |
| **Drag and drop** in a container view | The dragging delegate methods — `tableView(_:pasteboardWriterForRow:)` and the equivalents on `NSCollectionView` / `NSOutlineView`. Never call `beginDraggingSession(...)` by hand. |
| **Select text** outside an `NSTextView` | `NSTextSelectionManager` **(new in macOS 27)** — attach it to a view plus a selection data source for bidirectional selection, text drag-and-drop, and toggling. Gate on `if #available(macOS 27, *)`. |

### Modern dragging delegate

Create a pasteboard item, set its data, return it — AppKit drives the drag:

```swift
func tableView(
    _ tableView: NSTableView,
    pasteboardWriterForRow row: Int
) -> (any NSPasteboardWriting)? {
    let item = NSPasteboardItem()
    item.setString(viewModel.rows[row].id, forType: .string)
    return item
}
```

## Control events — target/action, no subclass

For user-driven tracking state on **standard controls** (buttons, sliders), register a target/action for a control event instead of writing tracking logic. **No subclassing required.**

```swift
let button = NSButton()
button.setAccessibilityIdentifier("refreshButton")          // every interactive control
button.addTarget(self, action: #selector(refresh),
                 for: .trackingEndedOutside)
```

For more control, add a standard `NSGestureRecognizer`; for maximum flexibility, subclass one. Verify the recognizer subtype with `sdk-api` first. UI types are `@MainActor` (Swift 6 strict concurrency); the action selector runs on the main actor.

## "My control won't respond to clicks" → overlapping sibling

Gesture recognizers operate on a view **and its subviews**. An overlapping sibling silently swallows the click before it reaches the control underneath.

- **Fix first:** resize the overlapping view so it no longer covers the control. With Auto Layout (the repo's default — see `appkit-design`), this is usually a missing or wrong constraint.
- **If the overlay must stay** (intentional), override `hitTest(_:)` to fall through:
  ```swift
  override func hitTest(_ point: NSPoint) -> NSView? { nil }
  ```

## Keyboard navigation — make the window fully Tab-navigable

The **key view loop** is the order Tab / Shift-Tab cycle controls. To keep it correct automatically as views are added or removed, enable recalculation on the window:

```swift
window.autorecalculatesKeyViewLoop = true
```

If you don't set this, **you** own creating and maintaining the loop via `nextKeyView` — error-prone for a dynamic view tree. Every interactive control still needs its accessibility identifier (separate from the VoiceOver label) so `appkit-ui-testing` can reach it.

## Status items and keyboard focus

Keyboard navigation reaches into the menu bar. The right wiring depends on what the item does:

- **Shows a menu** — already behaves like a menu bar menu. Nothing to do.
- **Triggers an action** — set a target + action (and an image) on `NSStatusItem.button`; it fires on Return during keyboard navigation.
- **Custom view** — set the status item's `view`, then add a target + action.
- **Shows a custom window** — AppKit must know when that UI is active so focus behaves. Use the **expanded interface session API (new in macOS 27)** — not a raw `NSPanel` + `canBecomeKey` hack.

### Expanded interface session

Set the delegate when the item is created, then show/hide the window on the session callbacks and **cancel the session to request dismissal** (e.g. after the user picks an action). Verify these symbols with `sdk-api` — they are macOS 27:

```swift
@MainActor
final class StatusController: NSObject, NSStatusItemExpandedInterfaceDelegate {
    private let statusItem: NSStatusItem

    func statusItem(
        _ statusItem: NSStatusItem,
        didBegin session: NSStatusItemExpandedInterfaceSession
    ) { /* show window */ }

    func statusItemDidEndExpandedInterfaceSession(
        _ statusItem: NSStatusItem, animated: Bool
    ) { /* order window out */ }

    func userPickedAction() {
        // take the action, then request dismissal:
        statusItem.expandedInterfaceSession?.cancel()
    }
}
```

The session may also be canceled **for you** when focus moves elsewhere. If this could be a SwiftUI surface, `MenuBarExtra` does much of this automatically.

## Common mistakes — STOP if you reach for any of these

| Reaching for… | Do instead |
|---|---|
| A click gesture recognizer to track selection in a table/collection view | Observe `selected` or the selection delegate callbacks — the dedicated APIs |
| `beginDraggingSession(...)` by hand | The pasteboard-writer delegate methods; AppKit drives the session |
| Context menus from `rightMouseDown(with:)` | `defaultMenu` / `menu` / `menu(for:)` — also handles Control-click + accessibility |
| Hand-maintained `nextKeyView` for a dynamic tree | `window.autorecalculatesKeyViewLoop = true` |
| A raw `NSPanel` + `canBecomeKey` for a status-item window | `NSStatusItemExpandedInterfaceDelegate` |
| A guessed delegate signature or `@available` version | `sdk-search` the pattern, `sdk-api check` the symbol |

No force-unwrap, no force-try — the repo builds in Swift 6 language mode and optional-safety is enforced.

## Verifying the build

After wiring input, run the app surface through mise — Tuist generates the project (the `.xcodeproj` / `.xcworkspace` / `Derived` are gitignored):

```bash
mise run -C macOS generate          # regenerate after manifest/source changes
mise run -C macOS build             # build the AppKit app
mise run -C macOS launch:macos      # build + launch to exercise the input by hand
mise run -C macOS test:app          # app-target / UI tests
mise run -C macOS fmt               # swift-format, committed .swift-format (lineLength 100, 4-space)
```

Input wiring lives at the view edge — mark controllers `// SPEC: manual` when no cross-target behavioral contract applies. Behavior that *is* spec-provable belongs in the `Core` view model and runs via `mise run -C Core test` / `specify verify` (Swift Testing, `swift` report format, the `.spec(…)`/`.scenario(…)` traits from `TestSupport`'s `SpecTraits.swift` — the scenario id lives in the trait, never the test name).

## When to invoke a more specific skill

- Picking the control or laying out the window? → `appkit-design`
- Writing UI tests that drive these controls by identifier? → `appkit-ui-testing`
- Scaffolding the app or editing `Project.swift`? → `appkit-setup`
- Shared `@Observable` / `Observations` idioms? → `ios-development`
- About to write tests, or claim work is done? → `test-driven-development`, `verification-before-completion`
- Click won't fire / focus is wrong and you can't see why? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`

## HIG and first-party docs

When an input affordance is unclear, read the HIG section and `sdk-search` the pattern — don't vendor HIG files.

- [Inputs](https://developer.apple.com/design/human-interface-guidelines/inputs) · [Pointing devices](https://developer.apple.com/design/human-interface-guidelines/pointing-devices) · [Keyboards](https://developer.apple.com/design/human-interface-guidelines/keyboards) · [The menu bar](https://developer.apple.com/design/human-interface-guidelines/the-menu-bar)
- [AppKit reference](https://developer.apple.com/documentation/appkit) · [Handling mouse events](https://developer.apple.com/documentation/appkit/handling-mouse-events) · [Drag and drop](https://developer.apple.com/documentation/appkit/drag-and-drop) · [NSGestureRecognizer](https://developer.apple.com/documentation/appkit/nsgesturerecognizer) · [NSStatusItem](https://developer.apple.com/documentation/appkit/nsstatusitem)

The Xcode MCP bridge (build/run/inspect from the agent) is per-machine — it lives in your user/local config, not the committed `.mcp.json`, and is not projected by SpecKit. See `ios-development` for the setup.
