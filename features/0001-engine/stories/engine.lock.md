---
id: story.engine.lock
kind: story
depends-on: [domain.lock, story.engine.verify]
---

# Story: Write the acknowledgment lock

As the engine,
I want `specify lock` to be the single writer of the drift lock,
So that drift state is recorded deterministically and exactly once, only when a platform is genuinely green (D7).

# Acceptance Criteria

## Scenario 1: Verify-on-green writes the lock

<!-- id: scenario.engine.lock.writes-on-green -->

- Given `specify verify <platform>` passes every scenario of a spec
- Then it invokes `specify lock` to write `.speckit/lock/<platform>/<spec-id>` with the spec's content hash and per-scenario results

## Scenario 2: Lock is the only writer

<!-- id: scenario.engine.lock.sole-writer -->

- Given any other command runs (scan, drift, cover, parity)
- Then none of them write or mutate a lock shard — only `specify lock` does

## Scenario 3: A red verify writes no lock

<!-- id: scenario.engine.lock.no-write-on-red -->

- Given `specify verify <platform>` has any failing or unjoinable scenario
- Then no lock shard is written for that spec (the prior shard, if any, is left untouched)

## Scenario 4: Lock edits are blocked

<!-- id: scenario.engine.lock.generated-guard -->

- Given a user or agent tries to hand-edit a file under `.speckit/lock/`
- When the generated-file gate runs (`story.engine.gate`)
- Then the edit is refused — the lock is engine-owned
