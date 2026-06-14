---
id: story.greeting.greet
kind: story
---

**As a** service consumer
**I want** a greeting endpoint
**So that** the scaffold's spec → test → verify loop is proven end to end

**Independent test:** `GET /greeting/Ada` returns `Hello, Ada!`

# Acceptance Criteria

## Scenario 1: greeting a caller by name

<!-- id: scenario.greeting.greet.hello -->

- Given a caller named "Ada"
- When they GET /greeting/Ada
- Then the response is "Hello, Ada!"

## Scenario 2: an empty name defaults to world

<!-- id: scenario.greeting.greet.defaults-to-world -->

- Given no name
- Then the greeting defaults to "Hello, world!"
