---
id: story.todo.manage
kind: story
---

**As a** user
**I want** to manage a list of to-dos
**So that** the scaffold's spec → test → verify loop is proven end to end

**Independent test:** `mise run -C {{.Dir}} test` proves the two scenarios below
against the headless Core package.

# Acceptance Criteria

## Scenario 1: toggling a to-do flips its completion

<!-- id: scenario.todo.manage.toggle -->

- Given an un-done to-do
- When it is toggled
- Then it is done

## Scenario 2: adding a to-do appends it to the list

<!-- id: scenario.todo.manage.add -->

- Given an empty list
- When a to-do is added by label
- Then the list contains one item with that label
