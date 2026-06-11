---
id: story.engine.scan
kind: story
depends-on: [domain.specmodel, conventions]
---

# Story: Scan the spec library

As a developer or agent working in a SpecKit repo,
I want `specify scan` to validate the whole spec library against the model invariants,
So that a malformed spec is caught before it forks across platforms or feeds a bad join.

# Acceptance Criteria

## Scenario 1: A well-formed library scans clean

<!-- id: scenario.engine.scan.clean -->

- Given a spec library where every spec satisfies invariants I1–I6 of `domain.specmodel`
- When the user runs `specify scan`
- Then the command exits 0
- And with `--json` it emits an empty `findings` array

## Scenario 2: A dangling depends-on is reported

<!-- id: scenario.engine.scan.dangling-depends-on -->

- Given a spec whose `depends-on` lists an `id` that no spec in the library declares
- When the user runs `specify scan`
- Then the command exits non-zero
- And a finding names the offending file, cites invariant I5, and identifies the unresolved `id`

## Scenario 3: A filename/ID mismatch is reported

<!-- id: scenario.engine.scan.filename-id-mismatch -->

- Given a spec whose `id` trailing segment does not equal its filename stem
- When the user runs `specify scan`
- Then the command exits non-zero
- And a finding cites invariant I1 with both the `id` and the filename

## Scenario 4: A duplicate ID is reported

<!-- id: scenario.engine.scan.duplicate-id -->

- Given two specs that declare the same `id`
- When the user runs `specify scan`
- Then the command exits non-zero
- And a single finding cites invariant I4 and lists both file paths

## Scenario 5: A story scenario missing its sub-ID is reported

<!-- id: scenario.engine.scan.missing-scenario-id -->

- Given a `story` spec containing a Gherkin scenario with no `<!-- id: scenario.* -->` declaration
- When the user runs `specify scan`
- Then the command exits non-zero
- And a finding cites invariant I6 and names the scenario heading
