---
id: story.engine.gate
kind: story
depends-on: [domain.specmodel, story.engine.drift]
---

# Story: Enforcement gate

As a maintainer,
I want `specify gate` to run honesty subchecks from pre-commit and CI,
So that enforcement is agent-agnostic and stronger than Claude-only hooks (D8).

# Acceptance Criteria

## Scenario 1: The test-edit firewall blocks an untethered test change

<!-- id: scenario.engine.gate.test-edit-firewall -->

- Given a commit changes a scenario-tagged test but does not change the spec that owns the scenario
- When `specify gate` runs
- Then it fails with the test path and the spec it should have accompanied

## Scenario 2: Generated files are protected

<!-- id: scenario.engine.gate.generated-block -->

- Given a commit edits a generated path (e.g. under `.speckit/lock/`, codegen output)
- When `specify gate` runs
- Then it fails, naming the generated path

## Scenario 3: Scoped-commit subject is required

<!-- id: scenario.engine.gate.scoped-commit -->

- Given a commit whose subject scope is not a defined scope (a spec id, a `features/<slug>` dir, one of the fixed harness areas, `specs`, `treewide`, or a scope declared in `.claude/commit-scopes`)
- When `specify gate` runs
- Then it rejects the subject and explains the scope rule

## Scenario 4: Agent-agnostic invocation

<!-- id: scenario.engine.gate.agent-agnostic -->

- Given any environment (a git pre-commit hook, a CI job, no agent at all)
- When `specify gate <subcheck>` is invoked
- Then it runs the same check with the same result, depending on no agent runtime (D8)

## Scenario 5: Each subcheck is independently runnable

<!-- id: scenario.engine.gate.subchecks -->

- Given the subchecks (test-edit firewall, generated-file block, scoped-commit, lint-on-dirty, drift-on-PR)
- When one is invoked by name
- Then only that check runs, so hooks and CI can compose them à la carte
