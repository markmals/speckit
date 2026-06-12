---
name: systematic-debugging
description: Use when encountering any bug, test failure, or unexpected behavior, before proposing or trying any fix. Find the root cause first; symptom fixes are failure. Pairs with test-driven-development (the regression test) and verification-before-completion (the gate).
---

# Systematic Debugging

Random fixes waste time and create new bugs. Quick patches mask underlying issues. Always find the root cause **before** attempting a fix.

## The Iron Law

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

If you haven't completed Phase 1, you cannot propose a fix.

## When to use

Any technical issue: test failures, bugs in dev or prod, unexpected behavior, performance problems, build failures, a `specify verify` join that won't go green, a `specify drift`/`parity` row you can't explain.

**Especially when:**

- Under time pressure (emergencies make guessing tempting; that's when this matters most).
- "Just one quick fix" seems obvious.
- You've already tried 1–2 fixes and they didn't work.
- You don't fully understand the issue.

**Don't skip when** the issue seems simple (simple bugs still have root causes) or you're in a hurry (systematic is faster than thrashing).

## The four phases

Complete each phase before proceeding to the next.

### Phase 1: Root cause investigation

**Before attempting any fix:**

1. **Read the error carefully.** Don't skim. Stack traces, line numbers, error codes, the failing assertion's expected-vs-actual — they often contain the exact answer. For a red `specify verify`, read *which* scenario is unbound or failing, not just that it's red.
2. **Reproduce consistently.** Can you trigger it reliably? Exact steps? If you can't reproduce, you can't verify a fix — gather more data instead of guessing.
3. **Check recent changes.** `git log`, `git diff`. New dependencies, config edits, a spec amendment, an environment difference. The cause is usually a recent change.
4. **For multi-component systems, gather evidence at each boundary.** When the bug spans test runner → report → join, or API → service → DB:
   ```
   For each boundary:
     - Log what data enters this layer
     - Log what data exits
     - Verify env / config propagation
     - Note state at each layer
   ```
   Run once. Read the evidence to find the failing layer. Then investigate **that** layer.
5. **Trace data flow backward.** Where does the bad value originate? What called this with it? Keep tracing up to the source. Fix at the source, not the symptom.

You leave Phase 1 with a clear, evidence-backed hypothesis about what is broken and where.

### Phase 2: Pattern analysis

1. **Find a working example.** Similar code elsewhere in this repo — or the same spec's implementation on a sibling target (`rg "SPEC: <id>"`) — that works.
2. **Read it completely.** Don't skim. Don't "adapt the pattern" — understand it.
3. **Identify differences.** List every difference between working and broken, however small. Don't assume "that can't matter."
4. **Understand dependencies.** What other components, settings, or environment does the working code rely on?

You leave Phase 2 knowing what's different between working and broken.

### Phase 3: Hypothesis and minimal test

1. **State the hypothesis.** "I think X is the root cause because Y." Be specific.
2. **Test minimally.** Smallest possible change to test it. **One variable at a time.**
3. **Verify.** Confirmed? → Phase 4. Not confirmed? → new hypothesis. Don't pile fixes on top.
4. **When you don't know:** say "I don't understand X yet." Don't pretend. Ask, research, or trace further.

### Phase 4: Implementation

1. **Write a failing test** — the simplest reproduction of the bug. Use `test-driven-development`; if the bug is a missed spec scenario, tag the regression test with the scenario's source binding so `specify verify` covers it going forward.
2. **Verify it fails for the right reason.** Same red-green discipline as TDD.
3. **Implement a single fix.** One change. No "while I'm here" improvements, no bundled refactors.
4. **Verify.** Test passes? Suite still green? `specify verify <platform>` green and re-locked? Use `verification-before-completion`.

### When 3+ fixes have failed: question the architecture

Pattern indicating an architectural problem:

- Each fix reveals a new shared-state / coupling problem in a different place.
- Fixes require "massive refactoring" to implement.
- Each fix creates a new symptom elsewhere.

**Stop attempting fixes.** This is not a failed hypothesis — it's a wrong architecture. Surface to the user: what you tried, what each fix revealed, the pattern, and your hypothesis about what's structurally wrong. Discuss before attempting another fix.

## Red flags — stop and return to Phase 1

"Quick fix for now, investigate later" · "Just try changing X and see" · "Add multiple changes, run tests" · "Skip the test, I'll manually verify" · "It's probably X, let me fix that" · "I don't fully understand but this might work" · "The pattern says X but I'll adapt it differently" · "One more fix attempt" (after 2+ failures). All mean: **stop, return to Phase 1.**

## User signals you're doing it wrong

"Is that not happening?" (you assumed without verifying) · "Will it show us…?" (add evidence-gathering) · "Stop guessing" (you're fixing without understanding) · "Question the fundamentals" (you're fixing symptoms) · "We're stuck?" (your approach isn't working). When you see these: **return to Phase 1.**

## Common rationalizations

| Excuse | Reality |
| --- | --- |
| "Issue is simple, don't need the process" | Simple issues have root causes too. The process is fast for simple bugs. |
| "Emergency, no time" | Systematic is faster than guess-and-check thrashing. |
| "Just try this first, then investigate" | The first fix sets the pattern. Do it right from the start. |
| "Multiple fixes at once saves time" | Can't isolate what worked. Causes new bugs. |
| "Reference too long, I'll adapt" | Partial understanding guarantees bugs. Read it completely. |
| "I see the problem, let me fix it" | Seeing symptoms ≠ understanding root cause. |
| "One more fix" (after 2+ failures) | 3+ failures = architectural problem. Discuss before attempting another. |

## Commit

Once the failing test passes, the fix is verified, and the suite is green, commit under the scoped-commits convention (`specify gate scope` enforces it). The subject names the user-observable bug (`fix: <bug>`); the body explains the **root cause** uncovered in Phase 1, not the symptom. Split into `test:` + `fix:` when the regression test is independently valuable. One root cause, one commit — never bundle a "while I'm here" cleanup or a second bug.

## Related skills

- `test-driven-development` — for the failing-test step in Phase 4.
- `verification-before-completion` — the gate before claiming the bug is fixed.
- `adversarial-review` — after the fix, when the surrounding code now "looks correct," try to break it again.
