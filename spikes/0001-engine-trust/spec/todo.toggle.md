---
id: story.todo.toggle
kind: story
---

# Story: Toggle a todo

As a user, I want to toggle a todo's done state, so that I can track progress.

# Acceptance Criteria

## Scenario 1: Completing an active todo

<!-- id: scenario.todo.toggle.complete -->

- Given an active todo
- When the user toggles it
- Then it becomes complete

## Scenario 2: Reactivating a completed todo

<!-- id: scenario.todo.toggle.reactivate -->

- Given a completed todo
- When the user toggles it
- Then it becomes active

## Scenario 3: Guarding against an empty label

<!-- id: scenario.todo.toggle.guard-empty -->

- Given a todo whose label is empty
- When the user toggles it
- Then the toggle is rejected
