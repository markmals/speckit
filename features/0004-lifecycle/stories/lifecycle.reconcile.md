---
id: story.lifecycle.reconcile
kind: story
depends-on: [story.lifecycle.apply]
---

# Story: Reconcile a platform that raced ahead

As a developer,
I want `reconcile <platform>` to fold a platform's direct change back into the spec and the other platforms,
So that a fix or change made straight in code becomes the shared contract instead of silent drift.

# Acceptance Criteria

## Scenario 1: Proposes spec updates from the lead platform

<!-- id: scenario.lifecycle.reconcile.proposes-spec -->

- Given a platform whose implementation/tests changed behavior beyond what the spec says
- When the user runs `reconcile <platform>`
- Then it reads that platform's impl + tests, diffs against the spec, and proposes spec updates

## Scenario 2: Proposes cross-platform updates

<!-- id: scenario.lifecycle.reconcile.proposes-others -->

- Given the spec is updated to match the lead platform
- When `reconcile` runs
- Then it proposes the corresponding updates to the other platforms' implementations and tests

## Scenario 3: Nothing lands without human approval

<!-- id: scenario.lifecycle.reconcile.human-approves -->

- Given proposed spec and cross-platform diffs
- When they are presented
- Then each is applied only after a human approves it — reconciliation is never automatic (CONVENTIONS)

## Scenario 4: A corrected bug is not a reconciliation

<!-- id: scenario.lifecycle.reconcile.bugfix-not-reconcile -->

- Given a platform change that merely *corrects* the platform to match the existing spec (a bug fix)
- When deciding what to do
- Then it is handled by `verify`, not `reconcile` — reconcile is for *changed* behavior the spec doesn't yet state
