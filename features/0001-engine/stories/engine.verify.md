---
id: story.engine.verify
kind: story
depends-on: [domain.specmodel, story.engine.scan]
---

# Story: Verify a target against the specs

As a developer or agent,
I want `specify verify <target>` to run the target's tests and join the results back to scenarios,
So that "green" provably means "the right scenarios were proven," and the result is recorded for drift.

# Acceptance Criteria

## Scenario 1: A passing target verifies green and writes the lock

<!-- id: scenario.engine.verify.green-writes-lock -->

- Given a target whose tagged tests cover and pass every scenario of the specs it implements
- When the user runs `specify verify <target>`
- Then the command exits 0
- And a lock shard is written at `.speckit/lock/<target>/<spec-id>` recording the spec's current content hash and per-scenario results

## Scenario 2: An unjoinable scenario fails loudly (D12)

<!-- id: scenario.engine.verify.unjoinable-scenario-fails -->

- Given a scenario that has no test tagged with its `[scenario.<id>]`
- When the user runs `specify verify <target>`
- Then the command exits non-zero
- And it reports the unjoinable scenario id explicitly
- And it does not report that scenario as passing, and does not write a green lock for its spec

## Scenario 3: A dangling test scenario reference fails loudly (D12)

<!-- id: scenario.engine.verify.dangling-test-ref -->

- Given a test tagged with a `[scenario.<id>]` that no story declares
- When the user runs `specify verify <target>`
- Then the command exits non-zero
- And it reports the dangling reference rather than silently ignoring the test

## Scenario 4: Outcomes join by test identity; the binding comes from source

<!-- id: scenario.engine.verify.source-bound-join -->

- Given a target whose report identifies each test by suite/class + name (v1: Vitest junit for web; Swift Testing's `--event-stream-output-path` NDJSON for apple) but does **not** carry the scenario ID (spike 0001)
- When the user runs `specify verify <target>`
- Then the engine reads each test's scenario binding from source (the `.scenario(…)` trait / `// [scenario.<id>]` comment) and joins it to the report outcome by test identity
- And per-scenario pass/fail is produced regardless of which runner emitted the report

## Scenario 5: A test with no source binding is a hard error

<!-- id: scenario.engine.verify.unbound-test -->

- Given a test that runs but carries no scenario binding in source
- When the user runs `specify verify <target>`
- Then the engine reports it as a hard error rather than silently dropping it (D12)
- And likewise a binding that points at a scenario the spec does not declare is a hard error
