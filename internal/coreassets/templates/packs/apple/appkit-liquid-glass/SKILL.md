---
name: appkit-liquid-glass
description: Use when adopting macOS 26/27 Liquid Glass in this repo's AppKit surface — `NSGlassEffectView`, the interactive glass effect that bounces on click, scroll edge effects, sidebar/toolbar glass, or making a custom view's corners concentric with its container via `NSViewCornerConfiguration` / `.containerConcentric`. Targets modern AppKit (macOS 26/27). Complementary to `appkit-design` (the broader UI skill), `appkit-setup` (scaffold), and `ios-development` (shared idioms).
---

# AppKit Liquid Glass (macOS 26/27)

This skill covers **the Liquid Glass material and concentric corners** in the AppKit surface under `macOS/Sources/App`. It is the deep dive on the material; `appkit-design`'s "Adopt Liquid Glass" step is the index — start there for picking a control or sizing a window, come here for the glass mechanics. The spec-provable domain and `@Observable` view models live in the headless `Core` package; glass is pure view edge — mark these sites `// SPEC: manual`. Liquid Glass arrived in macOS 26 and keeps evolving in 27, so the symbols here are **new** — the grounding mandate is not optional.

> **These are the symbols your memory is most likely to invent.** `NSGlassEffectView.effectIsInteractive` is the literal property that motivated building `sdk-api` — its name and min-macOS are exactly what a confident model guesses wrong. Verify every glass symbol and `@available` version before you write it.

## The grounding mandate (do this first, every time)

Before writing any Liquid Glass code, ground it with the two CLIs from the apple-platform-tools monorepo (`mise run install` → `~/.local/bin`; never vendor them):

```bash
sdk-search "interactive glass effect" "scroll edge effect" "concentric corners"   # one query per feature
sdk-api check NSGlassEffectView.effectIsInteractive                               # exists? + min macOS
```

Front-load the searches, read the canonical patterns, verify every symbol with `sdk-api check`, then write — gating each on the `@available` version `sdk-api` reports. Knowing a symbol exists is not verifying it; these are macOS 26/27-only and the min-macOS is the gate.

## Automatic vs. opt-in

If you adopted Liquid Glass in macOS 26, building against the macOS 27 SDK gives you these **for free** — no code:

- `NSScrollEdgeEffectStyle` resolves to a **hard-edge effect** automatically when free-floating text (like the title-bar title) sits over scrolling content.
- **Sidebars** extend to the window's edges, content flows behind them, and selection uses a semi-bold text style for emphasis.
- **Bordered toolbar items** over the sidebar adopt the glass look.

The one thing you **opt into in code** is the **interactive glass effect** (new in macOS 27): glass that subtly bounces when clicked, so a control feels physically responsive to interaction.

## Interactive glass — sparingly, on controls only

`NSGlassEffectView.effectIsInteractive` (`Bool`, macOS **27.0**) turns on the bounce. Restrict it to **controls and buttons, or glass containers of interactive controls** — never every glass surface. A little goes a long way; blanket interactivity reads as noise.

```swift
if #available(macOS 27, *) {
    glass.effectIsInteractive = true   // sdk-api-verified: NSGlassEffectView, 27.0
}
```

The glass view itself is `NSGlassEffectView`: assign your content to its `contentView` and set `cornerRadius` — never `addSubview`. To merge adjacent glass shapes into one continuous material, wrap them in `NSGlassEffectContainerView`. Both are macOS **26.0**; verify before writing.

## Concentric corners (`NSViewCornerConfiguration`)

Content near a container's corner should adopt the container's shape instead of fighting the window. **The closer a view sits to the corner, the more its radius should match.** Audit custom views that hardcode `layer.cornerRadius` near a window or split-view edge — that is exactly where concentricity belongs.

`cornerConfiguration` is a **read-only** property — override the getter, never assign it. Use `.containerConcentric(_:)` on `NSViewCornerRadius` to compute the radius from the container, passing a **minimum** so every corner is always rounded:

```swift
final class LocalWeatherView: NSView {
    let minimumCornerRadius: CGFloat = 8

    override var cornerConfiguration: NSViewCornerConfiguration? {
        let radius: NSViewCornerRadius = .containerConcentric(minimumCornerRadius)
        return .uniformCorners(radius: radius)   // same radii on all four corners
    }
}
```

`.uniformCorners(radius:)` keeps all four corners equal; pick a different factory if the design needs per-corner radii. `NSViewCornerConfiguration` is macOS **27.0** — gate `if #available(macOS 27, *)` and verify with `sdk-api`.

## Scroll edge, sidebar, and toolbar glass

- **Scroll-edge fade** — `preferredScrollEdgeEffectStyle` on a titlebar / split-view accessory controller (macOS **26.1**). The hard-edge resolution over free-floating text is automatic on 27; the explicit style is for tuning.
- **Sidebar** — stays an `NSSplitViewItem(sidebarWithViewController:)` (see `appkit-design`); glass changes the material, not the structure. Let content flow behind it; don't fake the edge-extension by resizing frames.
- **Toolbar** — a real, populated `NSToolbar` via its delegate. Bordered items over a sidebar pick up glass automatically; an empty toolbar renders nothing.

`.inset` is a *table style* and `NSVisualEffectView` is *vibrancy* (10.10) — **neither is adopting Liquid Glass**. Don't assert that a sidebar or an inset table already gives you the look; reach for `NSGlassEffectView` and gate it explicitly.

## Carry the design non-negotiables into glass code

Glass is still UI — the `appkit-design` hygiene rules apply unchanged:

- **Accessibility identifier on every interactive glass control** (`setAccessibilityIdentifier("…")`) — distinct from the VoiceOver label; `appkit-ui-testing` queries the identifier.
- **Semantic colors only** for any content over glass (`.labelColor`, `.controlAccentColor`, …) — never literal RGB; glass amplifies a hardcoded color's failure under Dark Mode / Increase Contrast.
- **Semantic typography** — `NSFont.preferredFont(forTextStyle:)`, never `ofSize:`.
- **Content-derived sizing** — derive from `fittingSize` / Auto Layout; never a magic frame.
- **Respect Reduce Transparency** — `NSWorkspace.shared.accessibilityDisplayShouldReduceTransparency` falls glass back to an opaque material; honor it.
- **Swift 6 strict concurrency** — glass views are `@MainActor`; no force-unwrap, no force-try.

## Common mistakes

- **Hardcoding `layer.cornerRadius` near a container corner** — that's a `cornerConfiguration` + `.containerConcentric` site; audit those first.
- **Interactive glass everywhere** — restrict `effectIsInteractive` to interactive controls or their glass containers.
- **Guessing a macOS 26/27 symbol or its min-macOS** — `sdk-api check` costs a second; inventing `effectIsInteractive`'s version ships a build that fails on the deployment target.
- **`addSubview` on an `NSGlassEffectView`** — assign to its `contentView` instead.

## Verifying the build

```bash
mise run -C macOS generate          # Tuist regenerates the project (.xcodeproj/.xcworkspace/Derived gitignored)
mise run -C macOS build             # build the AppKit surface
mise run -C macOS launch:macos      # launch to eyeball the glass on a real window
mise run -C macOS fmt               # swift-format, the committed .swift-format (lineLength 100, 4-space)
```

Glass is view edge with no cross-target spec; mark controllers `// SPEC: manual`. View-model and domain specs live in `Core` (`mise run -C Core test` / `specify verify`, `swift` report format, `.spec(…)`/`.scenario(…)` traits from `TestSupport`'s `SpecTraits.swift` — scenario id in the trait, never the test name; see `ios-development`).

## When to invoke a more specific skill

- Picking a control, sizing a window, the broader design hygiene? → `appkit-design`
- Scaffolding or a missing prerequisite (`sdk-api`/`tuist` not found)? → `appkit-setup`
- Writing UI tests that query these identifiers? → `appkit-ui-testing`
- Shared `@Observable` / `Observations` / Swift Testing idioms? → `ios-development`
- About to write tests, or claim work is done? → `test-driven-development`, `verification-before-completion`
- Debugging unexpected rendering? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`

## HIG and first-party docs

When a glass choice is unclear, read the HIG section and `sdk-search` the pattern — don't vendor HIG files.

- [Materials](https://developer.apple.com/design/human-interface-guidelines/materials) · [Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines)
- [AppKit reference](https://developer.apple.com/documentation/appkit) · [`NSGlassEffectView`](https://developer.apple.com/documentation/appkit/nsglasseffectview) · [`NSVisualEffectView`](https://developer.apple.com/documentation/appkit/nsvisualeffectview)

Source: WWDC 2026 Session 289, "Modernize Your AppKit App." The Xcode MCP bridge is per-machine — see `ios-development`'s MCP section (user/local config, not committed or projected by SpecKit).
