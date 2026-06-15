---
name: appkit-app-inspector
description: Use when learning how a running macOS app builds its UI — drive the flexscope runtime inspector (dev-box only) to read another app's live NSView tree, fonts, layers, and constraints, then translate the findings into your own public, semantic AppKit code. Covers the doctor precondition gate, the filter→drill restraint, the dev-box safety posture, and turning runtime facts into a recipe verified with sdk-api/sdk-search.
---

# AppKit App Inspector (flexscope)

Drive **flexscope** — a macOS runtime inspector that injects into a running app and exposes its real view tree (class names, frames, fonts, `CALayer` facts, Auto Layout constraints) over a CLI — to answer "how did they build *that*?", then rebuild the **fact** in your own AppKit code. flexscope is a separate, **dev-box-only** tool; this pack never vendors it. For *what to build* see the spec; for the canonical control once you know the answer, see `appkit-design`; for writing the macOS surface under `macOS/Sources/App`, see `ios-development`.

**The discipline is the whole point.** A capable agent already knows roughly which verb to call. What it skips — every time — is the dev-box safety framing and the filter→drill restraint. This skill makes both non-negotiable.

## Dev-box reality — state this plainly, unprompted

flexscope works by **injecting code into another process**, which requires turning off system security. Say this up front, every session:

- **A dedicated dev box only.** SIP **+** AMFI **+** library-validation off is a real, **system-wide** security regression — *any* process can load code into any other. The machine must hold **no real data or credentials**. (Reversible from Recovery.) Refuse to run on a normal machine.
- **The tool never ships.** The injected dylib / framework / `flexscope` CLI only work on a defanged box and are an attack tool elsewhere. They must **never** reach a Tuist target, a release build, a committed entitlement, or CI.
- **Only knowledge crosses into your product** — a font, a row height, a constraint, a material — **never the tool or the injection step.**
- **Don't inspect your own app with it.** Use a debugger you own. flexscope is for apps you *can't* debug.

## Step 1 — `doctor` gate (always first)

```bash
flexscope doctor
```

Runs locally, no injection. Six checks, fixed order — **`sip`, `amfi`, `libval`, `arm64e-abi`, `arch`, `flexmac-built`** — and *all six* must pass (`ok:true`, exit `0`). Any failure → exit `6` with a per-check `remedy`.

**On non-green: STOP.** Report exactly which check failed and its `remedy` **verbatim**; do **not** proceed to attach; do **not** change SIP/AMFI yourself. SIP-off alone is necessary but **not** sufficient — AMFI (`amfi_get_out_of_my_way=0x1`) is the real gate, and a plain-arm64 dylib (vs. `arm64e`) fails dyld **silently**. Remediation is the explicit `flexscope doctor --fix` flag's job; even then a reboot-pending box stays at exit `6` until a fresh `doctor` returns `0`.

## Step 2 — Filter → drill (never full-dump)

A single unbounded `tree` on a live Mail window is ~200k tokens you can't afford — and the answer was always a handful of nodes. **Locate few → project narrow → read deep on survivors:**

```bash
flexscope attach com.apple.mail                                              # pid or bundle id
flexscope windows com.apple.mail                                             # cheapest entry: window roots
flexscope find com.apple.mail --where 'class ~ NSTableRowView' --count-only  # size it BEFORE paying
flexscope find com.apple.mail --where 'title *= "Inbox"' --fields node,class,frame --limit 3
flexscope tree com.apple.mail --at <node> --depth 2 --fields class,frame     # shallow walk to the fact
flexscope font com.apple.mail --at <survivor>                                # deep-read ONE survivor
flexscope layer com.apple.mail --at <survivor>                               # the look: cornerRadius, filters, shadow
flexscope constraints com.apple.mail --at <survivor>                         # Auto Layout, both directions + intrinsic
```

**The rule:** a `--count-only` or narrow `find` **precedes** any deep read; `--depth` is always small; `--fields` projects only what picks the next branch. flexscope evaluates selectors **inside the target process** — only matching, projected nodes cross the wire; the CLI never pulls the tree to grep locally. Total traffic is a few KB, not a tree dump.

Selectors honor the **runtime class hierarchy** (`class ~ NSControl` matches an `NSButton`) — something the Accessibility tree can't do, and the headline reason flexscope exists. Predicate operators: `=` `!=` `*=` (substring) `~` (class-of) `matches` (regex) `> < >= <=` `and` `or` `not` `intersects`.

## Verb cheat-sheet

| Question | Verb | Notes |
|----------|------|-------|
| Preconditions OK? | `doctor` | gate; all-6-green or stop |
| What can I attach to? | `list-apps [--match …]` | pid · name · bundleId |
| Open/close a session | `attach <app>` / `detach <app>` | `<app>` = pid or bundle id |
| Top-level windows | `windows <app>` | cheapest entry; screen coords |
| Structure skim | `tree --at N --depth D --fields …` | depth-bounded; descend only `truncated` branches |
| Locate few | `find --where EXPR --fields … --limit N` / `--count-only` | server-side; 0 matches = exit 0, broaden |
| Deep-read one node | `node --at N --include class,frame,font,constraints,layer,ivars` | the drill target |
| Typography | `font --at N` / `fonts [--at N]` | `fonts` = de-duped subtree inventory |
| The look | `layer --at N` | cornerRadius, backgroundFilters, shadow (CALayer facts only) |
| Vibrancy material | `node --at N` | `material` / `blendingMode` are **node** fields, not `layer` output |
| Auto Layout | `constraints --at N` | both directions + hugging/compression + intrinsic |

Read `flexscope --help` / its schema for exact flags and unspecified defaults — **don't invent them.**

## Exit codes — branch on these, don't parse prose

`0` ok (a 0-match query is **still 0** — read `_meta.totalMatched`, broaden, don't re-issue) · `2` usage/bad selector (fix the query) · `3` app not running (launch + retry) · `4` not attached / injection failed (re-attach; injection-failed → read message: arch vs AMFI/LV) · `5` stale node (re-walk with `find`/`tree` for a fresh handle; never re-deref) · `6` precondition (report remedy, stop) · `7` timeout (retry once) · `8` schema mismatch (rebuild so CLI + dylib agree).

## SwiftUI boundary — don't over-claim

When a node carries `swiftUIBoundary:true`, everything below it is the AppKit/`CALayer` scaffold SwiftUI emits — **not** hand-written controls. Report the fonts, frames, fills, layers, and constraints you observe **confidently**; do **not** assert the underlying classes are hand-coded `NSView`s a developer would write.

## Step 3 — Translate to a recipe (the deliverable)

The output is **not** a tree dump — it's a **few-KB AppKit recipe**: the real control (or its public equivalent) rebuilt with **public, semantic** APIs. If a finding is a private class (`_NSTextFieldSimpleLabel`), reproduce the *behavior* with the public control, not the private name. **Verify before you write** — flexscope tells you what the *other* app did; it does not tell you the symbol is public or available on your deployment target:

- `sdk-api check NSColor.controlAccentColor` — confirm the symbol exists and its macOS availability before you type it.
- `sdk-search "vibrancy NSVisualEffectView material"` — find the canonical public pattern for the look you read off the wire.

Bake the runtime facts back through this pack's design non-negotiables (the inspector is how you *learn* them, not a license to skip them):

- **Semantic colors only** — `NSColor.labelColor`, `.secondaryLabelColor`, `.controlAccentColor`, `.separatorColor`. A `layer.backgroundColor` you read as raw RGBA maps to a *semantic* token, never a hardcoded literal.
- **Semantic typography** — `NSFont.preferredFont(forTextStyle:)`. A `17pt` you read off a label is a *text style*, not a magic point size.
- **Content-derived sizing** — reproduce the constraints / `fittingSize` you observed with Auto Layout; never copy the frame as a magic `NSRect`.
- **Liquid Glass** — adopt it explicitly through the public material/effect APIs `sdk-search` surfaces; don't reconstruct a private vibrancy hack.
- **Accessibility identifier on every interactive control**, separate from its label — so the recipe is drivable from `appkit-ui-testing` even though the inspected app's identifier was incidental.
- **Swift 6 strict concurrency**, `@MainActor` on the UI type, no force-unwrap / force-try.

Land the recipe in `macOS/Sources/App`, prove its view-model in the headless `Core` package, and verify with `mise run -C macOS build` / `test` and `specify verify`. Pair with `appkit-design` for the canonical control and `ios-development` for the surrounding view-controller idioms.

## Distribution advisory (reciprocal)

Most findings translate to clean public AppKit. But if you reproduce something that **requires** a private API, surface the App-Store-review / Developer-ID trade-off unprompted: review may reject case-by-case; Developer ID + notarization is the escape hatch. Prefer the public equivalent `sdk-search` finds.

## Getting flexscope

flexscope is a separate repo (not part of this scaffold, not projected by `specify init`), built with SwiftPM (`swift build -c release`) and code-signed for injection on the dev box. Clone and build it per its own README, then run `flexscope doctor` to confirm the box is ready. The CLI contract is frozen; some flag defaults are unspecified — read `--help` / the schema rather than guessing.

## When to invoke a more specific skill

- Picking the canonical public control for a finding? → `appkit-design`
- Writing the macOS surface around the recipe? → `ios-development`
- Asserting the rebuilt UI's behavior? → `appkit-ui-testing`
- Debugging an injection / exit-code surprise? → `systematic-debugging`
- About to claim the recipe is done? → `verification-before-completion`

Note: Xcode's external-agent MCP is per-machine and lives in your user/local config — see `ios-development` for the setup; this skill doesn't re-cover it.
