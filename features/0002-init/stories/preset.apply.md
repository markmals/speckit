---
id: story.preset.apply
kind: story
depends-on: [story.init.basic, story.extension.add]
---

# Story: Apply a preset

As a developer,
I want `specify init --preset <id>` (or `specify preset apply <id>`) to install a curated bundle in one step,
So that a known-good combination of extensions and config lands without my wiring it by hand.

A preset is a named bundle resolved from the catalog; applying it installs its extensions and overlays its config.

# Acceptance Criteria

## Scenario 1: Apply a preset during init

<!-- id: scenario.preset.apply.during-init -->

- Given `specify init --integration claude --preset <id>`
- Then the preset's extensions are installed and its config overlaid as part of init
- And the resulting install state lists the preset and each extension it brought in

## Scenario 2: Apply a preset to an existing project

<!-- id: scenario.preset.apply.after-init -->

- Given an initialized project
- When the user runs `specify preset apply <id>`
- Then the preset's extensions/config are applied additively, the same as during init

## Scenario 3: An unknown preset gives a clear error

<!-- id: scenario.preset.apply.not-found -->

- Given a preset id that the catalog does not contain
- When the user applies it
- Then the command exits non-zero with a clear message and installs nothing

## Scenario 4: A preset's extensions remove cleanly

<!-- id: scenario.preset.apply.reversible -->

- Given a preset was applied
- When its extensions are removed
- Then each restores prior state per `story.extension.remove`, leaving no preset residue
