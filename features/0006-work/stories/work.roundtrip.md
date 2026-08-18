---
id: story.work.roundtrip
kind: story
depends-on: [domain.work-item]
---

# Story: Five verbs drive every provider

As a developer or agent coordinating work,
I want `specify work ready/create/claim/move/list` to behave identically on every provider,
So that switching providers changes storage, never workflow.

# Acceptance Criteria

## Scenario 1: Create lands in ready

<!-- id: scenario.work.roundtrip.create-lands-ready -->

- Given a configured work provider
- When the user runs `specify work create <title>`
- Then the item appears in `ready`

## Scenario 2: Claim moves to in-progress

<!-- id: scenario.work.roundtrip.claim-moves-to-in-progress -->

- Given an item in `ready`
- When the user runs `specify work claim <id>`
- Then the item's state is `in-progress`

## Scenario 3: Move reaches any canonical state

<!-- id: scenario.work.roundtrip.move-to-state -->

- Given an existing item
- When the user runs `specify work move <id> <state>` for any canonical state
- Then `specify work list --state <state>` reflects the move

## Scenario 4: List without a state returns everything

<!-- id: scenario.work.roundtrip.list-all -->

- Given items across several states
- When the user runs `specify work list`
- Then every item is returned

## Scenario 5: A defect is created by type

<!-- id: scenario.work.roundtrip.defect-type -->

- Given `specify work create <title> --type defect`
- Then the item records the `defect` type — this is what former defect intake became

## Scenario 6: A spec pointer is recorded

<!-- id: scenario.work.roundtrip.spec-pointer -->

- Given `specify work create <title> --spec <spec-id>`
- Then the item records the spec pointer
