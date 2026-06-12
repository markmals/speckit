---
name: test-driven-development
description: Use when writing any production code — features, bug fixes, refactors, behavior changes. Write the failing test first, watch it fail, write minimal code to pass, refactor green. Each test is bound to its scenario per the platform's source-binding convention.
---

# Test-Driven Development

Write the test first. Watch it fail. Write the minimal code to pass it. Refactor while green.

**Core principle:** if you didn't watch the test fail, you don't know if it tests the right thing.

## The Iron Law

```
NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST
```

If you wrote code before the test, **delete it** and start over. Don't keep it as "reference". Delete means delete.

## When to use

**Always:** new features (every spec implementation), bug fixes (write a failing regression test first), behavior changes, refactors that change observable behavior.

**Exceptions (ask first):** generated code, configuration, throwaway prototypes that will be deleted.

If you're thinking "skip TDD just this once" — stop. That's rationalization.

## Red → Green → Refactor → Lock

```
1. RED      Write one failing test for one behavior, bound to its scenario.
2. Verify   Run it. Confirm it FAILS — for the right reason.
3. GREEN    Write the minimal code to make it pass.
4. Verify   Run it. Confirm it passes; confirm other tests still pass.
5. Refactor Clean up while staying green.
6. LOCK     `specify verify <platform>` — joins the test to its scenario and
            records the spec as verified-green (the lock) on this platform.
7. Repeat   Next failing test for the next behavior.
```

### RED — write one failing test

- One behavior per test.
- **Bind it to its scenario in source**, using the platform's native affordance — never by polluting the test description. See `specs/CONVENTIONS.md`:
  - **Swift Testing:** `@Test(.scenario("scenario.items.list.empty"))` on a raw-identifier func.
  - **kotlin.test:** `@Tag("scenario:items.list.empty")`.
  - **MSTest:** `[TestProperty("scenario", "items.list.empty")]`.
  - **Vitest / Rust / Go:** a `// [scenario.items.list.empty]` comment directly above the test.
- Test real code; mock only what you can't control (network, time, randomness).
- **Example or invariant?** A `domain.<entity>` spec's **Invariants** section that says "always / never / for all" earns a **property-based test**, not just hand-picked examples — `adversarial-review` hunts exactly the inputs your examples skipped. Per-platform runners: fast-check (web), SwiftCheck / `@Test(arguments:)` (Apple), kotest-property (Android), FsCheck (C#), proptest (Rust). Keep the example tests too; the property is the net underneath them.

### Verify RED — watch it fail

**Mandatory. Never skip.** Run the tests. Confirm it **fails** (a failing assertion, not an undefined symbol), the message matches what you expect, and it fails because the **feature is missing** — not a typo. If it passes immediately, you're testing existing behavior; fix the test.

### GREEN — minimal code

The simplest code that makes the failing test pass. No defensive branches for untested cases, no options/hooks for future flexibility (YAGNI), no "while I'm here" refactors.

### Verify GREEN — watch it pass

It passes, all other tests still pass, and the output is pristine — no warnings, no leaked logs.

### Refactor, then Lock

Clean up while green (no new behavior), then run `specify verify <platform>`: it runs the suite, joins each result to its scenario, and writes the lock for the specs that are fully green. From then on `specify drift`/`cover`/`parity` track that fact. A scenario with no bound test, or a test bound to a scenario that doesn't exist, **fails the verify loudly** — fix the binding.

## Why order matters

Tests written after the implementation **pass immediately** — proving nothing. They might test the wrong thing, test what you implemented rather than what's required, or miss edge cases. Test-first forces you to **see the test fail**, which proves the test actually tests something.

## Common rationalizations

| Excuse | Reality |
| --- | --- |
| "Too simple to test" | Simple code still breaks. The test takes 30 seconds. |
| "I'll test after" | Tests passing immediately prove nothing. |
| "Already manually tested" | Manual ≠ automated. No record, can't re-run. |
| "Deleting work is wasteful" | Sunk cost. Keeping unverified code is debt. |
| "My examples cover the invariant" | Examples cover the inputs you thought of. "For all" means the ones you didn't — write the property. |

## Red flags — stop and start over

Code before the test · test added "after we ship" · test passes immediately · can't explain why the test failed · "just this once" · "keep as reference, write tests, then adapt". All of these mean: **delete the code, start with TDD.**

## Verification checklist

Before marking work complete: every behavior has a test · every "always / for all" invariant has a property test · watched each test fail for the expected reason · minimal code to pass · all tests pass · output pristine · real code (mocks only when unavoidable) · each test is **bound to its scenario** in source · `specify verify <platform>` is green and the lock is written.

Can't tick every box? You skipped TDD. Start over.

## Commit

Each green-refactor-lock cycle is a natural commit boundary. Commit once green (per `specs/CONVENTIONS.md` scoped-commit discipline; `specify gate scope` enforces it). **Never commit while red.** One commit per behavior when test and impl are tightly bound; split into `test:`-then-fix when the failing test is valuable on its own.

## Related skills

- `verification-before-completion` — the gate before claiming done.
- `adversarial-review` — tries to defeat the tests this skill wrote.
- `systematic-debugging` — when a test fails for a reason you don't yet understand.
