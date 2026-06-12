---
id: story.welcome.greet
kind: story
---

**As a** developer
**I want** the app to greet a user by name
**So that** the scaffold's spec → test → verify loop is proven end to end

**Independent test:** `greeting("Ada")` returns `"Hello, Ada!"`

# Acceptance Criteria

## Scenario 1: greeting a user

<!-- id: scenario.welcome.greet.hello -->

- Given a user named "Ada"
- When the app greets them
- Then the greeting reads "Hello, Ada!"
