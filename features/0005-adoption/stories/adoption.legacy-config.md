---
id: story.adoption.legacy-config
kind: story
depends-on: [story.adoption.target-add]
---

# Story: An unmigrated config still runs

As a developer whose project adopted an earlier SpecKit,
I want a config written by that version to keep loading,
So that upgrading the binary never bricks the project.

# Acceptance Criteria

## Scenario 1: Retired keys are ignored with one notice

<!-- id: scenario.adoption.legacy-config.retired-keys-ignored -->

- Given a config carrying keys this version retired (`stack`, `deploy`)
- When any command loads it
- Then the keys are ignored
- And a single notice names them and points at MIGRATION.md — never a hard failure

## Scenario 2: An older schema version loads

<!-- id: scenario.adoption.legacy-config.older-version-loads -->

- Given a config declaring an older schema version
- When any command loads it
- Then it loads and every engine command works

## Scenario 3: Writes land at the current schema version

<!-- id: scenario.adoption.legacy-config.rewritten-current -->

- Given a command that writes the config
- When it writes
- Then the config is written at the current schema version
