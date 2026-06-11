---
id: story.extension.remove
kind: story
depends-on: [story.extension.add]
---

# Story: Remove an extension and restore prior state

As a developer,
I want `specify extension remove` to cleanly uninstall an extension and restore whatever it overrode,
So that install/remove round-trips and the project is never left in a half-overridden state (D3).

# Acceptance Criteria

## Scenario 1: Remove uninstalls the extension's artifacts

<!-- id: scenario.extension.remove.uninstall -->

- Given an installed extension
- When the user runs `specify extension remove <id>`
- Then its projected commands/skills and its install-state entry are removed

## Scenario 2: Remove restores an overridden command

<!-- id: scenario.extension.remove.restore-override -->

- Given extension A overrode a command previously provided by B (by priority)
- When A is removed
- Then B's version of the command is restored from the recorded override state

## Scenario 3: Add→remove is an identity round-trip

<!-- id: scenario.extension.remove.round-trip -->

- Given a project at a known state
- When an extension is added and then removed
- Then the project returns to its prior state (file tree and install state both)

## Scenario 4: Removing something not installed is a clear no-op error

<!-- id: scenario.extension.remove.not-installed -->

- Given an id that is not installed
- When the user runs `specify extension remove <id>`
- Then the command reports it is not installed and changes nothing
