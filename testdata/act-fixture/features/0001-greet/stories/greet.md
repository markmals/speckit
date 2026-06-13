---
id: story.greet
kind: story
---

**As a** maintainer of the gate Action
**I want** a tiny end-to-end SpecKit project
**So that** the gate self-test exercises a real scenario↔test join on a runner

**Independent test:** `greet("Ada")` returns `"Hello, Ada!"`

# Acceptance Criteria

## Scenario 1: greeting a user

<!-- id: scenario.greet.hello -->

- Given a user named "Ada"
- When the helper greets them
- Then the greeting reads "Hello, Ada!"
