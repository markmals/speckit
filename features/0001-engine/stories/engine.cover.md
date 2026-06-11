---
id: story.engine.cover
kind: story
depends-on: [story.engine.verify, domain.specmodel]
---

# Story: Coverage for a spec

As a developer or agent,
I want `specify cover <spec-id>` to show which platforms implement a spec and which of their tests pass,
So that I can see, per spec, where the work stands across the platform matrix.

# Acceptance Criteria

## Scenario 1: Shows per-platform coverage

<!-- id: scenario.engine.cover.per-platform -->

- Given a spec implemented on some platforms and not others
- When the user runs `specify cover <spec-id>`
- Then each applicable platform is listed with its state: implemented-and-green, implemented-but-red, or missing

## Scenario 2: Green is read from the lock, not re-run

<!-- id: scenario.engine.cover.reads-lock -->

- Given platforms with existing lock shards
- When the user runs `specify cover <spec-id>`
- Then "green" is derived from each platform's lock (last verified-green at the current spec hash), without re-running tests

## Scenario 3: Distinguishes drifted from green

<!-- id: scenario.engine.cover.drifted -->

- Given a platform whose lock hash no longer matches the spec
- When the user runs `specify cover <spec-id>`
- Then that platform shows as drifted, not green

## Scenario 4: JSON output

<!-- id: scenario.engine.cover.json -->

- Given `specify cover <spec-id> --json`
- Then it emits a structured per-platform record (platform, state, scenarios covered)
