---
id: story.engine.cover
kind: story
depends-on: [story.engine.verify, domain.specmodel]
---

# Story: Coverage for a spec

As a developer or agent,
I want `specify cover <spec-id>` to show which targets implement a spec and which of their tests pass,
So that I can see, per spec, where the work stands across the target matrix.

# Acceptance Criteria

## Scenario 1: Shows per-target coverage

<!-- id: scenario.engine.cover.per-target -->

- Given a spec implemented on some targets and not others
- When the user runs `specify cover <spec-id>`
- Then each target with lock state is listed with its state — conforming, drifted, or missing — derived from the lock without re-running tests (a "red" state would require a fresh verify, which scenario 2 forbids)

## Scenario 2: Green is read from the lock, not re-run

<!-- id: scenario.engine.cover.reads-lock -->

- Given targets with existing lock shards
- When the user runs `specify cover <spec-id>`
- Then "green" is derived from each target's lock (last verified-green at the current spec hash), without re-running tests

## Scenario 3: Distinguishes drifted from green

<!-- id: scenario.engine.cover.drifted -->

- Given a target whose lock hash no longer matches the spec
- When the user runs `specify cover <spec-id>`
- Then that target shows as drifted, not green

## Scenario 4: JSON output

<!-- id: scenario.engine.cover.json -->

- Given `specify cover <spec-id> --json`
- Then it emits a structured per-target record (target, state, scenarios covered)
