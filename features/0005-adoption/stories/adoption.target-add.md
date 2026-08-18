---
id: story.adoption.target-add
kind: story
depends-on: [story.engine.verify]
---

# Story: Register an existing target

As a developer adopting SpecKit into a repo whose code already exists,
I want `specify target add` to record my target in config as one command,
So that adoption renders no files, runs no scripts, and never asks what my project is built with.

# Acceptance Criteria

## Scenario 1: Registering existing code touches only the config

<!-- id: scenario.adoption.target-add.registers-existing-code -->

- Given a repo with existing code and tests
- When the user runs `specify target add <name> --dir <path> --format <format> --report <path> --source <path> --command <shell>`
- Then the target is recorded in `.speckit/specs.json`
- And no file is created or modified other than that config
- And no script is run

## Scenario 2: The target directory must exist

<!-- id: scenario.adoption.target-add.requires-existing-dir -->

- Given a `--dir` that does not exist
- When the user runs `specify target add`
- Then the command fails naming the missing directory
- And writes nothing

## Scenario 3: Errors never speak platform vocabulary

<!-- id: scenario.adoption.target-add.no-platform-vocabulary -->

- Given any invalid invocation
- When the command fails
- Then the error names only the missing or invalid flag
- And no error or help text asks the user to choose a platform, stack, or scaffold

## Scenario 4: An unknown report format is rejected

<!-- id: scenario.adoption.target-add.rejects-unknown-format -->

- Given a `--format` outside `junit` | `swift` | `gotest`
- When the user runs `specify target add`
- Then the command fails listing the known formats

## Scenario 5: Multiple source paths are all recorded

<!-- id: scenario.adoption.target-add.multi-source -->

- Given `--source` repeated
- When the user runs `specify target add`
- Then all of the paths are recorded
- And the engine scans each of them for bindings
