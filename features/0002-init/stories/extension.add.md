---
id: story.extension.add
kind: story
depends-on: [story.init.basic]
---

# Story: Add an extension

As a developer,
I want `specify extension add` to install an extension from the catalog, a URL, or a local path,
So that I can layer capability onto a project after init, additively and reversibly (D3).

Behaviors mined from the oracle's `tests/test_extensions.py` (D14).

# Acceptance Criteria

## Scenario 1: Add a bundled extension by id

<!-- id: scenario.extension.add.bundled -->

- Given a bundled extension id
- When the user runs `specify extension add <id>`
- Then the extension's commands/skills are projected for the project's agent and recorded in install state

## Scenario 2: Add by display name resolves to the id

<!-- id: scenario.extension.add.by-display-name -->

- Given an extension referenced by its display name
- When the user runs `specify extension add "<Display Name>"`
- Then the name is resolved to the canonical id and that id is used for install and state

## Scenario 3: An unknown extension gives a clear error

<!-- id: scenario.extension.add.not-found -->

- Given an id that matches no bundled or catalog extension
- When the user runs `specify extension add <id>`
- Then the command exits non-zero with a clear "not found" message naming the id, not a stack trace

## Scenario 4: Add from a URL prompts before doing work

<!-- id: scenario.extension.add.from-url -->

- Given `specify extension add --from <url>`
- Then the user is prompted to confirm before any download/spinner begins
- And cancelling at the prompt exits cleanly with no partial install

## Scenario 5: Dev install links a local extension

<!-- id: scenario.extension.add.dev -->

- Given `specify extension add --dev <local-path>`
- Then the local extension is linked (symlinked) into the project
- And where symlinks are unavailable (e.g. Windows) it falls back to a copy
- And `--force` reinstalls over an existing install

## Scenario 6: Priority is honored on install

<!-- id: scenario.extension.add.priority -->

- Given two extensions that project the same command
- When one is added with a higher `--priority`
- Then its version of the command wins, and the override is recorded so removal can restore the loser
