---
id: story.adoption.reference-target
kind: story
depends-on: [story.adoption.target-add]
---

# Story: The reference target is configuration

As a developer with several targets,
I want the target that others are matched against to be a config key,
So that no platform is hardcoded as the reference and a single-target project needs no ceremony.

# Acceptance Criteria

## Scenario 1: A configured reference target is reported

<!-- id: scenario.adoption.reference-target.configured -->

- Given `reference_target` names a defined target
- When the engine reports the reference
- Then that target is reported as the reference

## Scenario 2: A sole target is the reference

<!-- id: scenario.adoption.reference-target.sole-target-is-reference -->

- Given exactly one target and no explicit `reference_target`
- When the engine reports the reference
- Then that target is the reference — nothing else could be

## Scenario 3: Unset privileges nothing

<!-- id: scenario.adoption.reference-target.unset-privileges-nothing -->

- Given several targets and no `reference_target`
- When the engine reports the reference
- Then no target is the reference
- And the engine privileges none

## Scenario 4: The key must name a defined target

<!-- id: scenario.adoption.reference-target.must-name-a-target -->

- Given `reference_target` naming a target that is not defined
- When the user runs `specify scan`
- Then the undefined reference is reported
