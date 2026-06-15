---
name: appkit-launch-continuity
description: Use when this repo's macOS AppKit app blocks system shutdown/restart while a sheet is up, won't quit cleanly, or loses its open windows/selection/arrangement after quit + relaunch. Covers graceful termination (preventsApplicationTerminationWhenModal) and state restoration (NSWindowRestoration). Targets modern macOS; complementary to `appkit-design` (the window/controller surface) and `ios-development` (shared idioms).
---

# AppKit Launch Continuity: Graceful Quit + State Restoration

**A great Mac app quits without pushback and comes back as if it was never gone.** People quit whenever they want — sometimes the system does it for them (an overnight reboot for an update). The app should block quit *only* when it genuinely must, and on relaunch restore exactly where the user left off: open windows, the selected item, and frontmost/minimized/full-screen state. This is polish on the AppKit surface under `macOS/Sources/App` (the `AppDelegate` + window controllers); the spec-provable domain stays in headless `Core`. Source: WWDC 2026 Session 289, "Modernize Your AppKit App."

## Ground it first

This is AppKit code, so the grounding mandate from `appkit-design` applies unchanged — verify every symbol and its `@available` version before you write it, and search the canonical pattern. Don't answer from memory; the restoration APIs are exactly where invented selectors and wrong `@MainActor` isolation slip in.

```bash
sdk-search "window state restoration" "graceful termination"   # one query per feature
sdk-api check NSWindow.preventsApplicationTerminationWhenModal  # exists? + min macOS
sdk-api check NSWindowRestoration.restoreWindow
```

## When to reach for this

- The app blocks a system restart/shutdown because a sheet or modal is up.
- After quit + relaunch the app loses its windows, selection, or arrangement.

## Part 1 — Graceful termination

A window presenting a sheet may not be able to close — and if a window can't close, the app can't quit. One property decides it:

```swift
window.preventsApplicationTerminationWhenModal = false
```

- It **defaults to `true`**, for good reason: it protects unsaved data (a "Save this document?" sheet that genuinely needs an answer).
- Set it to **`false`** on every sheet or modal that does **not** strictly require user intervention — inspectors, non-blocking dialogs. That lets the app terminate gracefully.

Decide it per-modal based on whether the modal is critical. You do **not** need to inspect the quit reason (`kAEQuitReason` Apple Events, `applicationShouldTerminate:` archaeology) — that's the over-engineered path. Set the one property.

## Part 2 — State restoration (`NSWindowRestoration`)

Three steps: **opt in → encode UI state → decode to restore windows and UI.** The window controllers and the restoration handler live under `macOS/Sources/App`; the `AppDelegate` owns the controllers the handler reaches for.

### Step 1 — Opt in (in the window controller)

```swift
@MainActor
final class MainWindowController: NSWindowController, NSWindowDelegate {
    convenience init() {
        let window = NSWindow(/* ... derive size from content, never a magic frame */)
        window.identifier = NSUserInterfaceItemIdentifier(WindowIdentifiers.mainWindow)
        window.setFrameAutosaveName(WindowIdentifiers.mainWindow)
        window.isRestorable = true
        window.restorationClass = WindowRestorationHandler.self
        self.init(window: window)
    }
}
```

- **`identifier`** — stable identity for the window (a code-stable token, not localized).
- **`setFrameAutosaveName`** — for *common* windows (main, preferences); restores them to the same space with the same frame. **Not needed for document windows.** (`appkit-design` already sets this when sizing a window to its content — same name, one source of truth.)
- **`isRestorable = true`** — lets AppKit call `encodeRestorableState`/`restoreState` and auto-restore which window was minimized, frontmost, and full-screen.
- **`restorationClass`** — invoked on relaunch to recreate the window.

### Step 2 — Encode UI state

```swift
override func encodeRestorableState(with coder: NSCoder) {
    super.encodeRestorableState(with: coder)   // always call super
    coder.encode(
        selectedProduct?.identifier.uuidString,
        forKey: RestorationKeys.productIdentifier)
}
```

- **Encode UI state only.** The goal is to reconstruct the *UI*, not re-serialize the app — never encode data that lives in your `Core` model, document, or database. Encode the identifier of the selection, then re-resolve it from `Core` on restore.
- All `NSResponder`s have `encodeRestorableState` — override it on views too where a view holds restorable UI state.
- It's called **only when state has been invalidated.** Whenever a view-hierarchy change should alter saved state, call `invalidateRestorableState()`:

```swift
splitViewController.onProductSelected = { [weak self] _ in
    self?.invalidateRestorableState()
}
```

AppKit then calls `encodeRestorableState` on everything invalidated, before quit. Forget this call and `encodeRestorableState` never fires — nothing is saved.

### Step 3 — Restore on relaunch (windows first, then state)

**Restore windows** in the restoration class — called for *every* window being restored. It's an `NSObject` conforming to `NSWindowRestoration`; pin it `@MainActor` since it touches the `AppDelegate` and AppKit windows:

```swift
@MainActor
final class WindowRestorationHandler: NSObject, NSWindowRestoration {
    static func restoreWindow(
        withIdentifier identifier: NSUserInterfaceItemIdentifier,
        state: NSCoder,
        completionHandler: @escaping @MainActor (NSWindow?, (any Error)?) -> Void
    ) {
        let delegate = NSApp.delegate as? AppDelegate
        switch identifier.rawValue {
        case WindowIdentifiers.mainWindow:
            completionHandler(delegate?.mainWindowController?.window, nil)
        case WindowIdentifiers.imageWindow:
            let controller = ImageWindowController()
            delegate?.imageWindowControllers.append(controller)
            completionHandler(controller.window, nil)
        default:
            completionHandler(nil, RestorationError.unknownWindow(identifier))
        }
    }
}
```

> **Always call the completion handler — AppKit waits on every restorable window.** If creation fails, call it with the error. If you can't call it inline, capture the handler and call it later, but *be certain you call it* — a missed call hangs relaunch.

**Then restore the UI** for each window with the same coder you encoded into:

```swift
override func restoreState(with coder: NSCoder) {
    super.restoreState(with: coder)
    if let productId = coder.decodeObject(
        of: NSString.self,
        forKey: RestorationKeys.productIdentifier) as String? {
        splitViewController?.selectProduct(id: productId)   // re-resolve against Core
    }
}
```

## Common mistakes — STOP if you catch yourself here

| Excuse / slip | Reality |
|---------------|---------|
| "I'll inspect the quit reason / `applicationShouldTerminate:`." | Over-engineered. Set `preventsApplicationTerminationWhenModal = false` on non-critical modals. |
| "I'll just encode the whole model into the coder." | Encode only what rebuilds the *UI* (a selection id); re-resolve data from `Core`. |
| "The completion handler is optional / I'll skip it on failure." | AppKit hangs waiting. Always call it — with the error if creation failed. |
| "Calling `super` is boilerplate." | Omitting `super` in `encodeRestorableState`/`restoreState` silently breaks the chain. |
| "Saving will happen on its own." | Without `invalidateRestorableState()`, `encodeRestorableState` never fires and nothing is saved. |

## Verify the build

After wiring restoration, run the app surface through mise (Tuist generates the project; `.xcodeproj`/`.xcworkspace`/`Derived` are gitignored):

```bash
mise run -C macOS build             # build the AppKit app
mise run -C macOS launch:macos      # launch, set a selection, ⌘Q, relaunch — does it come back?
mise run -C macOS test:app          # app-target tests
mise run -C macOS fmt               # swift-format (.swift-format: lineLength 100, 4-space)
```

State restoration is behavior you confirm by **doing** it: select an item, full-screen or minimize a window, quit, relaunch, and check it returns exactly. For graceful termination, quit while a non-critical sheet is up — it should not block. Window controllers carry `// SPEC: manual` (no cross-target behavioral contract); see `verification-before-completion` before claiming done.

## Related

- The window/controller surface this builds on, plus semantic color/typography and content-derived sizing → `appkit-design`.
- Shared `@Observable` / `Observations` / Swift Testing idioms, and the Xcode MCP bridge (per-machine, not committed) → `ios-development`.
- Process: `implementing-a-spec`, `test-driven-development`, `verification-before-completion`, `systematic-debugging`.
- Apple's code sample "Restoring your app's state with AppKit" and [developer.apple.com/documentation/appkit/nswindowrestoration](https://developer.apple.com/documentation/appkit/nswindowrestoration); HIG on [Modality](https://developer.apple.com/design/human-interface-guidelines/modality). Don't vendor docs — `sdk-search` the pattern.
