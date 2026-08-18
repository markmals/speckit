---
id: story.lifecycle.apply
kind: story
depends-on: [story.engine.verify, story.engine.drift, domain.ledger]
---

# Story: Apply a spec to a target

As a developer or agent,
I want `apply <spec-id> <target>` to regenerate a spec's tests and implementation on a target,
So that bringing a target in line with a spec is one command that ends in a verified-green lock (D9).

# Acceptance Criteria

## Scenario 1: Failing tests precede implementation

<!-- id: scenario.lifecycle.apply.tests-first -->

- Given a spec to apply on a target
- When `apply` runs
- Then it writes the scenario-tagged tests first and observes them fail, before writing the implementation that makes them pass (the `fail_first_observed` ledger signal)

## Scenario 2: Plan/tasks are disposable

<!-- id: scenario.lifecycle.apply.disposable-artifacts -->

- Given `apply` generates `plan.md`/`tasks.md` as disposable execution artifacts
- When the target reaches green and the lock is written
- Then those artifacts are deleted or archived — they are never the durable record (D9)

## Scenario 3: Ends in a verified-green lock

<!-- id: scenario.lifecycle.apply.green-lock -->

- Given `apply` completes successfully
- Then `specify verify <target>` passes for the spec and a lock shard is written (`story.engine.lock`)
- And `specify drift <target>` is clean for that spec immediately after

## Scenario 4: A wrong spec stops the loop

<!-- id: scenario.lifecycle.apply.spec-wrong -->

- Given that, while applying, the spec itself is found to be wrong or contradictory
- When `apply` detects it
- Then it stops and reports that the spec must be fixed, rather than encoding the contradiction into the implementation

## Scenario 5: Each attempt is recorded

<!-- id: scenario.lifecycle.apply.ledger -->

- Given any `apply` run (success or failure)
- Then exactly one ledger record is appended (`domain.ledger`, `story.lifecycle.ledger`)
