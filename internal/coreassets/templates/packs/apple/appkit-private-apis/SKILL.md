---
name: appkit-private-apis
description: Use when discovering, declaring, calling, or swizzling a private/undocumented AppKit or Objective-C symbol from the macOS app surface — bridging headers, `@objc protocol` + `unsafeBitCast`, `dlsym`, runtime class/selector lookup, and `method_exchangeImplementations`/`method_setImplementation` (idempotent install, restoration, thread-safety) — plus the App Store distribution trade-off you must surface unprompted.
---

# AppKit Private APIs

Discover a private AppKit/Objective-C symbol, declare a typed interface to it, call it from Swift, and — only when you must change framework behavior — swizzle it safely. Legitimate for research, debugging, internal tools, and Developer-ID apps. **The technique is the easy part; shipping it responsibly is the part that gets skipped.**

This skill *informs, it never gates.* It helps you use private APIs and tells you the trade-off so you decide with eyes open — surface the [distribution advisory](#distribution-advisory) **unprompted** whenever the code could ship. For *writing* the surrounding AppKit (windows, views, the app delegate), see `appkit-setup` and `appkit-design`. For driving the build, see the mise tasks below.

## Grounding mandate — applies double here

The pack rule is **never guess a symbol name or `@available` version — verify with `sdk-api`, search patterns with `sdk-search`** before writing AppKit code. Private symbols make this non-optional, because the public SDK won't catch your mistake:

- `sdk-api check NSWindow._setTitlebarSeparatorStyle` confirms whether a symbol is even *known*, and any macOS availability — but a **private** symbol carries no `@available` and no compiler check, so it can vanish between OS releases with zero warning.
- `sdk-search` surfaces the canonical *public* affordance first. **Always look for the supported API before reaching for a private one** — a public path you missed beats any swizzle.

`sdk-api`/`sdk-search` are external CLIs (built from apple-platform-tools, `mise run install` → `~/.local/bin`); never vendor them. They verify *existence and availability*; they do **not** clear a symbol for App Store review (see the advisory).

## The workflow

### 1. Look for a public API first

Run `sdk-search` for the behavior you want. If a supported AppKit affordance exists, use it and stop — no private API needed. Only proceed when the capability genuinely has no public surface.

### 2. Find the real selector

Recover the owning class, the exact selector, and its argument types from a header dump (PrivateHeaderKit reconstructs ObjC headers from the framework binaries — a *static* `xcrun` read, no SIP/AMFI changes; repo at <https://github.com/lynnswap/PrivateHeaderKit>, not vendored). A grep hit is a fact about *one* SDK build, not a contract — **verify it still exists at runtime** (step 4) before relying on it.

### 3. Declare the interface — pick the lightest mechanism that compiles

| You have | Declare via | Notes |
|----------|-------------|-------|
| ObjC method on a public class | **ObjC category** in a bridging header (`@interface NSWindow (Private)`) | Cleanest; fully type-checked |
| Pure-Swift target, one method | **`@objc protocol`** + `unsafeBitCast(obj, to:)` | No bridging header needed |
| A private **ivar** | `class_getInstanceVariable` + `object_getIvar` | Avoids the KVC string scanner |
| A private **C function** | **`dlsym`** on a `dlopen` handle → `@convention(c)` cast | Non-ObjC entry points |
| A private **class** you must not name with a literal | `NSClassFromString` (assembled at runtime) | Defeats the *static* scanner only |

`unsafeBitCast` does **no** conformance check, so a missing selector won't fault at the cast — it crashes at the call site. The guard in step 4 is mandatory.

### 4. Call it — guard every private touch, degrade never crash

Resolve classes with `NSClassFromString(_:)` and selectors with `NSSelectorFromString(_:)`. Gate every access so an OS change *degrades the feature* instead of crashing the app, and keep a public-API fallback. Probe once, cache the result; never re-`dlsym` on a hot path.

```swift
import AppKit
import ObjectiveC.runtime

@objc private protocol PrivateWindowAnimating {  // typed shape, no bridging header, no emitted runtime class
    func _setTransformForAnimation(_ t: CGAffineTransform, animate: Bool)
}

extension NSWindow {
    /// Returns false (and the caller falls back to public API) if the symbol moved.
    @discardableResult
    func tryPrivateTransform(_ t: CGAffineTransform, animate: Bool) -> Bool {
        let sel = NSSelectorFromString("_setTransformForAnimation:animate:")
        guard responds(to: sel) else { return false }      // OS dropped it → degrade
        unsafeBitCast(self, to: PrivateWindowAnimating.self)
            ._setTransformForAnimation(t, animate: animate)
        return true
    }
}

// Caller always keeps a public-API path:
if !window.tryPrivateTransform(t, animate: true) {
    window.animator().setFrame(target, display: true)
}
```

| Touching… | Guard with | On failure |
|---|---|---|
| a private **method** | `obj.responds(to: sel)` / `cls.instancesRespond(to:)` | public-API fallback |
| a private **class** | `NSClassFromString("X") != nil` | skip feature, log once |
| a private **ivar** | `class_getInstanceVariable(cls, "name") != nil` | treat as absent |
| a private **C symbol** | `dlsym(handle, "f") != nil` | no-op the path |

Never force-unwrap `NSClassFromString(...)!` or assume `responds(to:)` — an OS update *will* remove a private symbol eventually.

### 5. Swizzle — only when you must change existing behavior

Swizzling replaces a method's `IMP` process-wide; a half-swizzle corrupts **every instance** of the class. Do it correctly or not at all. Prefer `method_setImplementation` with a captured original `IMP` over `method_exchangeImplementations` — no extra selector polluting the class, and restoration is an exact pointer.

```swift
import ObjectiveC.runtime

enum WindowProbe {
    private static var originalIMP: IMP?
    private static let lock = NSLock()
    private static let selector = #selector(NSWindow.makeKeyAndOrderFront(_:))
    typealias MakeKeyFn = @convention(c) (AnyObject, Selector, AnyObject?) -> Void

    /// Idempotent, thread-safe install — safe from any number of call sites.
    static func install() {
        lock.lock(); defer { lock.unlock() }
        guard originalIMP == nil,                                   // one-time guard
              let m = class_getInstanceMethod(NSWindow.self, selector) else { return }
        let orig = method_getImplementation(m)
        originalIMP = orig
        let origFn = unsafeBitCast(orig, to: MakeKeyFn.self)
        let block: @convention(block) (AnyObject, AnyObject?) -> Void = { obj, sender in
            // ...your added behavior...
            origFn(obj, selector, sender)                          // RULE 1: call the original
        }
        method_setImplementation(m, imp_implementationWithBlock(block))
    }

    /// Restoration path — put the captured original back, exactly.
    static func uninstall() {
        lock.lock(); defer { lock.unlock() }
        guard let orig = originalIMP,
              let m = class_getInstanceMethod(NSWindow.self, selector) else { return }
        method_setImplementation(m, orig)                          // RULE 4: exact restore
        originalIMP = nil
    }
}
```

**The five non-negotiables:**

1. **Call the original.** Capture it (`method_getImplementation`, or the exchanged selector) and invoke it via a `@convention(c)` cast on every path — never drop it.
2. **Idempotent install.** A `static let`/`NSLock` guard so a double-install can't double-wrap or silently un-swizzle.
3. **Thread-safety.** Serialize the get-IMP→set-IMP under one lock; install before threads race the method, never swap a hot method live.
4. **Restoration path.** Stash the original `IMP`; provide an uninstall. "Not worth it" is unacceptable beyond a throwaway.
5. **Know when *not* to.** Subclass to override behavior you own; **KVO** for property changes (swizzling setters fights KVO's own isa-swizzling); proxy the **delegate** or use `NotificationCenter` for callbacks; `object_setClass` for one object. Reach for a class-wide swap last.

## Distribution advisory

Say this **unprompted** the moment private-API or swizzling code could end up in a shipped build. It is a *distribution* decision, not a correctness one. Inform; never gate — *"here's the trade-off and your options,"* never *"you may not."*

- **App Store review is case-by-case.** Apple judges private-API use case-by-case — no auto-reject, no public allow-list, and a past pass is no guarantee. Tell the user *"assume it might be rejected, and here's your fallback"* — never *"this will pass"* or *"this will be rejected."*
- **Scanner-evasion is not policy-safety.** Runtime-assembled class names, `object_getIvar` instead of KVC, `NSSelectorFromString` — these defeat *static* analysis only. Runtime instrumentation can still observe the call and reject the build. Evasion is a robustness tool (degrade gracefully across OS versions), not a compliance strategy. **Never frame it as "now it's safe to ship."**
- **Developer ID + notarization is the escape hatch.** Distributing outside the Mac App Store (web / Sparkle / direct, Developer-ID-signed and notarized) removes the review question — notarization scans for malware, it does not police private APIs. The trade you accept: an OS update may move a symbol and you own that breakage, so guard every call (step 4) and degrade. The app-store-connect deploy kind targets the App Store; a Developer-ID/notarize channel is the alternative for private-API apps.

## Anti-patterns

| ❌ Don't | ✅ Do |
|---------|------|
| Guess a private selector/class from memory | `sdk-search` for a public path first; dump + `sdk-api` verify; runtime-check before use |
| "Avoid the string literal → App-Store-safe" | Scanner-evasion ≠ policy-safe; surface case-by-case risk + Developer-ID hatch |
| Force-unwrap `NSClassFromString(...)!` / call unguarded | Gate every private touch; keep a public-API fallback |
| Swizzle and never call the original | Capture the `IMP`, invoke it via `@convention(c)` cast every path |
| Non-idempotent swizzle in `+load`/init | One-time `static let`/`NSLock` guard |
| "Restoring is too hard, I'll leave it" | Stash the original `IMP`; provide `uninstall()` |
| Swizzle to override behavior you own, or a hot method live | Subclass / KVO / delegate proxy; install at launch before first use |

## On the macOS app surface

Private-API code is `@MainActor` (it touches AppKit) under Swift 6 strict concurrency, lives behind a typed shim in `macOS/Sources/App`, and stays out of the headless `Core` package — `specify verify` proves the spec-provable domain, and private framework behavior is neither spec truth nor deterministically testable. Keep it isolated behind a capability check so the app degrades to public behavior when the symbol is gone. No force-unwrap, no force-try. Run `mise run -C macOS fmt`/`lint` (swift-format, `.swift-format`: lineLength 100, 4-space) before committing; build and launch with `mise run -C macOS build | launch:macos`.

## When to invoke a more specific skill

- Setting up windows / the app delegate / the public AppKit surface? → `appkit-setup`, `appkit-design`
- About to claim it works? → `verification-before-completion`
- A private call crashing or behaving oddly? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`, `test-driven-development`

First-party references: [AppKit](https://developer.apple.com/documentation/appkit) · [Objective-C runtime](https://developer.apple.com/documentation/objectivec/objective-c_runtime) · [Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines) (and `sdk-search`).
