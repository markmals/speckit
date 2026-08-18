---
id: story.work.markdown
kind: story
depends-on: [domain.work-item]
---

# Story: The markdown provider is the offline default

As a developer who wants work tracking with zero infrastructure,
I want the default provider to be one committed markdown file,
So that work state is diffable in review, versioned with the code, and needs no network and no external binary.

The file (default `WORK.md`) is a flat list under state headings:

```md
# Work

## Ready

- [ ] `wk-3` Write the parity docs · spec: story.engine.parity

## In Progress

- [ ] `wk-1` Wire the junit adapter

## Blocked

## Done

- [x] `wk-2` Fix the drift exit code
```

Each `## <Section>` heading is a state — its slug is the state name (`## In Progress` ↔ `in-progress`). Each item is one list line: a checkbox, a stable short id in backticks, the title, and an optional ` · spec: <spec-id>` suffix. The provider deliberately has **no dependency graph** — typed dependencies are the beads provider's job.

# Acceptance Criteria

## Scenario 1: Sections are states

<!-- id: scenario.work.markdown.sections-are-states -->

- Given the work file
- Then an `## <Section>` heading IS a state: its slug is the state name
- And an item's state is decided solely by which section it sits under

## Scenario 2: Short ids are stable and sequential

<!-- id: scenario.work.markdown.stable-short-ids -->

- Given items being created over time
- Then each item carries a stable short id in backticks
- And a new id is allocated as the next free `wk-<n>`

## Scenario 3: The spec pointer is an inline suffix

<!-- id: scenario.work.markdown.inline-spec-pointer -->

- Given an item created with `--spec <spec-id>`
- Then its line carries a ` · spec: <spec-id>` suffix

## Scenario 4: Done renders checked

<!-- id: scenario.work.markdown.done-is-checked -->

- Given items across states
- Then items in the done state render as `- [x]`
- And items in every other state render as `- [ ]`

## Scenario 5: Every verb works offline

<!-- id: scenario.work.markdown.offline -->

- Given no network and no external binary
- When any of the five verbs runs
- Then it succeeds

## Scenario 6: A missing file is an empty list

<!-- id: scenario.work.markdown.missing-file-is-empty -->

- Given a project with no work file
- When the user runs `specify work list`
- Then it lists nothing rather than failing

## Scenario 7: Parse and render round-trip

<!-- id: scenario.work.markdown.roundtrips -->

- Given any work file the provider wrote
- When it is parsed, rendered, and parsed again
- Then the result is stable — parse → render → parse is the identity
