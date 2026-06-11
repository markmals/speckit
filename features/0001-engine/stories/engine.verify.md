---
id: story.engine.verify
kind: story
depends-on: [domain.specmodel, story.engine.scan]
---

# Story: Verify a platform against the specs

As a developer or agent,
I want `specify verify <platform>` to run the platform's tests and join the results back to scenarios,
So that "green" provably means "the right scenarios were proven," and the result is recorded for drift.

# Acceptance Criteria

## Scenario 1: A passing platform verifies green and writes the lock

<!-- id: scenario.engine.verify.green-writes-lock -->

- Given a platform whose tagged tests cover and pass every scenario of the specs it implements
- When the user runs `specify verify <platform>`
- Then the command exits 0
- And a lock shard is written at `.speckit/lock/<platform>/<spec-id>` recording the spec's current content hash and per-scenario results

## Scenario 2: An unjoinable scenario fails loudly (D12)

<!-- id: scenario.engine.verify.unjoinable-scenario-fails -->

- Given a scenario that has no test tagged with its `[scenario.<id>]`
- When the user runs `specify verify <platform>`
- Then the command exits non-zero
- And it reports the unjoinable scenario id explicitly
- And it does not report that scenario as passing, and does not write a green lock for its spec

## Scenario 3: A dangling test scenario reference fails loudly (D12)

<!-- id: scenario.engine.verify.dangling-test-ref -->

- Given a test tagged with a `[scenario.<id>]` that no story declares
- When the user runs `specify verify <platform>`
- Then the command exits non-zero
- And it reports the dangling reference rather than silently ignoring the test

## Scenario 4: Reports normalize across runners

<!-- id: scenario.engine.verify.normalizes-reports -->

- Given a platform that emits a supported test report (v1: Vitest junit for web; Swift Testing's `--event-stream-output-path` NDJSON for apple — **not** `swift test --xunit-output`, which drops the scenario tag; spike 0001)
- When the user runs `specify verify <platform>`
- Then per-scenario pass/fail results are produced from that report regardless of which runner emitted it

## Scenario 5: A report that cannot carry the scenario tag is rejected

<!-- id: scenario.engine.verify.report-format-adequacy -->

- Given a report format that does not preserve the scenario tag (e.g. Swift Testing xunit, which emits only function names)
- When the verify adapter is configured to use it
- Then the engine refuses that source rather than silently reporting zero matched scenarios (D12)
- And it directs the adapter to a format that preserves the tag (the event stream)
