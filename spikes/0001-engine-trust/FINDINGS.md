# Spike 0001 — Findings

**Date:** 2026-06-11 · **Status:** complete · **Verdict:** gate is viable *with conditions*.

## Gate question

> Is mechanized parity + the scenario-to-test join trustworthy enough to gate a PR on, or does the deviation cell need a human in the loop?

**Answer: yes, parity can gate — *if and only if* (a) the join hard-fails on any unjoinable/dangling reference (D12), and (b) the deviation cell is never auto-greened (D11).** Concretely, the deviation cell *does* need a human: a `(deviates:)` marker over a *failing* test is indistinguishable, mechanically, from an honest one — so it must be surfaced as SUSPECT, and even an honest declared-deviation must be marked "needs sign-off," never green.

## Evidence

Real reports from real toolchains (Node 24 / Vitest, Swift 6.4 / Swift Testing), joined by the throwaway tool in `join/`:

```
== D12 — scenario↔test join ==
  ✗ JOIN ERROR (web): test references undeclared scenario scenario.todo.toggle.complete-typo (renamed/typo)

== Parity matrix ==
  scenario        web             apple
  complete        ✓ conforming    ✓ conforming
  guard-empty     ✓ conforming    ⚠ SUSPECT   (marker claims "native form prevents empty entry" but the test FAILED)
  reactivate      ✓ conforming    ~ declared-deviation (needs sign-off: "iOS reactivates via long-press")

== Gate verdict ==  → parity cannot auto-green this PR; needs human sign-off (exit 1)
```

## D12 — join failure modes discovered

1. **Vitest junit is directly joinable.** The scenario tag rides in the `<testcase name="story… > [scenario.x] …">` attribute; a regex recovers it, `<failure>` child = fail. Web adapter is easy.
2. **Swift Testing xunit is lossy — do NOT use it.** `swift test --xunit-output` (a) silently *rewrites the path* to `<base>-swift-testing.xml`, and (b) emits **function names** (`guardEmpty()`) in `name`, dropping the `@Test` display name — so the scenario tag is **gone**. A naive xunit join finds zero Apple scenarios and would report "no coverage" instead of failing.
3. **The native event stream is the Apple source of truth.** `swift test --event-stream-output-path <ndjson>` carries `test` records with `displayName` (scenario tag intact, hyphens and all) and `event`/`issueRecorded` records for outcomes. The join is clean off this.
4. **Dangling refs are caught.** A test tagged to a renamed/typo'd scenario (`complete-typo`) is detected as a hard JOIN ERROR — exactly the D12 "fail loudly, never silently zero-match" requirement.
5. **Function-name mangling is ambiguous** (the alternative to the event stream). Mangling `scenario.todo.toggle.guard-empty` → `scenario_todo_toggle_guard_empty` collapses `.` and `-` to the same `_`, so it can't be demangled unambiguously. Another reason to prefer the event stream over name-mangling for Swift.

## D11 — parity / deviation findings

1. **The four-cell model needs a fifth state: SUSPECT.** A `(deviates:)` marker on a scenario whose test *failed* must not be classified as `declared-deviation` (which reads green-ish). It is the adversarial case the marker is designed to hide. The engine computes deviation-presence and test-outcome **independently** and crosses them.
2. **Even honest deviations are "needs sign-off," not green.** `reactivate` legitimately deviates and its test passes — but the engine still can't verify the *reason* is valid, so the cell is advisory, not a pass. This is D11 confirmed empirically.
3. **Deviation-marker granularity is wrong in CONVENTIONS.** Markers are defined at the *spec* level (`// SPEC: <story-id> (deviates:)`), but parity is *scenario*-level. A story-level marker can't say "scenario 2 deviates, scenario 3 conforms." The spike used scenario-scoped markers (`// SPEC: scenario.x (deviates:)`) and they worked — CONVENTIONS should allow scenario scoping. (Amendment applied.)

## Recommendations for Phase 3

- **Apple verify adapter consumes the `--event-stream-output-path` NDJSON**, not `--xunit-output`. xunit is XCTest-era and lossy for Swift Testing. (Revisit once `xcodebuild`/xcresult is in play for UI tests, but for `swift test` the event stream wins.)
- **Web adapter parses Vitest junit `name` attributes** — sufficient as-is.
- **`specify parity` ships five states** (conforming / declared-deviation / drifted / missing / **suspect**); deviation cells are gated for sign-off and never auto-green.
- **CONVENTIONS allows scenario-scoped deviation markers** (applied this pass).
- The join must be a **typed, per-format adapter with a conformance test per format** (D12) — the formats differ enough that one regex won't do.

## Disposition

Throwaway. `web/`, `apple/`, `join/` stay parked here as reproducible evidence; nothing is promoted to `internal/`. Build artifacts (`node_modules/`, `.build/`) are gitignored; the small report files are kept as evidence.
