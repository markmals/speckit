---
name: appkit-hig
description: Use when a macOS/AppKit design decision turns on what the Apple Human Interface Guidelines actually say — the right control for a job, when to use a sheet vs a popover, an exact metric (hit-target size, layout margins, icon sizes), a platform convention, or verifying a design claim against Apple's guidance. Answer from the live HIG text and cite it; never recite from memory.
---

# Apple HIG (AppKit)

This skill is about **consulting the Apple Human Interface Guidelines** before you make a macOS design decision. For _how_ to build the decision in modern AppKit (symbol-verified code, Liquid Glass, semantic color/typography, window sizing), see `appkit-design`. For the macOS app scaffold and mise tasks, see `appkit-setup`. For the spec workflow, see `implementing-a-spec`.

**Core principle: when a question turns on a HIG rule, read the actual guideline and answer from it — do not recite the HIG from memory.** Memory is where the wrong number and the invented "best practice" come from. The authoritative answer is one fetch away, and HIG specifics (the _exact_ hit target, the _exact_ condition for a sheet vs a popover) are precisely what gets misremembered.

The HIG lives at [developer.apple.com/design/human-interface-guidelines](https://developer.apple.com/design/human-interface-guidelines). SpecKit does **not** vendor it — there is no offline corpus in this pack. You consult the live source.

## The grounding mandate

A question is HIG-grounded when it asks **what Apple recommends, requires, or specifies** — the right control for a job, when to use A vs B, an exact metric, a platform convention, or whether a design claim is actually in the guidelines. For any of those:

1. **Find the page.** Topics map to kebab-case slugs under the HIG root:
   - `…/human-interface-guidelines/buttons`, `…/menus`, `…/sheets`, `…/popovers`, `…/color`, `…/typography`, `…/sf-symbols`, `…/the-menu-bar`, `…/designing-for-macos`.
   - When the slug isn't obvious, `sdk-search "<pattern>"` surfaces the canonical AppKit pattern and the HIG section that backs it; or search the HIG site directly.
2. **Read the page with `WebFetch`** against the canonical URL — Apple publishes no `/llms.txt`, so fetch the doc page. Quote or paraphrase the **actual rule**, and **name the source** (e.g. "per the HIG _Buttons_ page…"). Link the URL so the user can follow up.

Do not skip step 1 because the answer "seems obvious." `44×44 pt, but 28×28 pt for pointer-driven macOS` is the kind of specific that's only right if you read it.

## Where things live

The HIG's top sections, each a browse index linking its children:

| Section | What's in it | URL |
| --- | --- | --- |
| **Getting started** | Per-platform design guides (`designing-for-macos`) + cross-platform principles | `…/human-interface-guidelines/designing-for-macos` |
| **Foundations** | Accessibility, color, dark mode, layout, materials, motion, typography, SF Symbols, icons, writing, inclusion | `…/human-interface-guidelines/foundations` |
| **Components** | The control catalog — buttons, menus, toolbars, sheets, popovers, split views, lists/tables, sidebars, pickers, text fields, windows, panels | `…/human-interface-guidelines/components` |
| **Patterns** | Task guidance — modality, drag and drop, entering data, feedback, launching, onboarding, settings, undo/redo, searching | `…/human-interface-guidelines/patterns` |
| **Inputs** | Control methods — keyboards, pointing devices, gestures, the menu bar | `…/human-interface-guidelines/inputs` |

For a worked-example shortlist of macOS affordances, see the HIG-driven affordances section in `ios-development`.

## HIG → AppKit, in this repo

The HIG settles **what** and **why**; it is prose guidance across all Apple platforms. To turn a settled decision into macOS code, hand off to `appkit-design`, which carries the macOS control mapping and verifies every symbol with `sdk-api`. A few of those handoffs the HIG keeps coming back to:

- **Liquid Glass** — the HIG _Materials_ guidance is explicit that the system material is adopted, not faked with hardcoded blurs. `appkit-design` shows the AppKit adoption.
- **Semantic color** — the HIG _Color_ and _Dark Mode_ pages require system/semantic colors so appearance and accessibility follow the system. In code that means `NSColor.labelColor`, `.controlAccentColor`, never a hardcoded RGB.
- **Semantic typography** — the HIG _Typography_ page maps to `NSFont.preferredFont(forTextStyle:)`, never magic point sizes.
- **Content-derived sizing** — HIG _Layout_ wants windows sized to their content; in AppKit that is `fittingSize` / Auto Layout, never a magic frame.
- **Accessibility** — the HIG _Accessibility_ page is the source for hit-target minimums and the requirement that every control is reachable and labeled. (The accessibility _identifier_ on every interactive control is separate from its visible label — see `appkit-ui-testing`.)

When the HIG names a control (e.g. "sidebar", "inspector"), the AppKit mapping (`NSSplitViewItem(sidebarWithViewController:)`, an inspector split-view item) lives in `appkit-design` — settle the choice here, implement it there.

## Provenance & staleness

You are reading the **live** HIG, so the wording is current — but it changes with each OS release. When a question is about a very recent change, fetch the page fresh rather than trusting a cached read from earlier in the session, and note the platform/OS the guidance targets. If `WebFetch` can't reach the page, say so and point the user at the URL rather than answering from memory.

## Red flags — STOP if you think any of these

| Thought | Reality |
| --- | --- |
| "I know what the HIG says about buttons/sheets/color." | Then fetching the page costs one `WebFetch` and proves it — including the exact number you're about to misquote. |
| "It's a quick design question, I'll just answer." | If it turns on a HIG rule, it's grounded. Find the page first. |
| "Close enough to the guideline." | An exact hit target or the exact sheet-vs-popover condition is only right if you read it. |
| "I'll cite the HIG without naming a page." | Name the page and link the URL. An ungrounded "per Apple's guidelines" is indistinguishable from a guess. |
| "I'll guess the AppKit symbol the HIG implies." | Verify it with `sdk-api check <Type.symbol>` and search patterns with `sdk-search` — that's `appkit-design`'s job. Never guess symbol names or `@available` versions. |

## When to invoke a more specific skill

- Implementing the decision in AppKit code? → `appkit-design`
- Setting up or generating the macOS app (Tuist, mise tasks)? → `appkit-setup`
- Accessibility identifiers and UI tests? → `appkit-ui-testing`
- About to claim the design is done? → `verification-before-completion`
- Implementing a spec end-to-end? → `implementing-a-spec`
