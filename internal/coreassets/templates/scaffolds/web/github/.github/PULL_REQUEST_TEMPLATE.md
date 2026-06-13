## What changed

<!-- The behavior this PR adds or fixes, and why. -->

## Spec integrity

<!-- SpecKit gates spec-honesty on PRs. The spec library is the source of truth and
     `specify verify <target>` writes the proof (the lock). Confirm the library and
     its test bindings stay in sync. -->

- [ ] Behavior change? Scenarios added/updated under `features/` or `specs/`.
- [ ] Each new/changed scenario is bound to a test (`// SPEC:` pointer / trait / title).
- [ ] `specify verify <target>` is green locally (the lock is written).
- [ ] `specify drift <target>` is clean (no spec drifted from its lock).
- [ ] No hand-edits to generated paths (`.speckit/lock/`, …).

## Linked defect

<!-- If this fixes a filed defect, link it: `Fixes #123`. The scenario + its lock
     are the durable proof — the issue is just intake, and closes on a green verify. -->
