---
id: story.cli.check
kind: story
depends-on: [conventions]
---

# Story: Check required tools

As an agent or developer,
I want `specify check` to report which required tools are available,
So that I can tell before starting work whether the toolchain a task needs is installed.

# Acceptance Criteria

## Scenario 1: Reports per-tool status

<!-- id: scenario.cli.check.reports -->

- Given a host with some tools installed and others not
- When the user runs `specify check`
- Then each checked tool is listed with a found/missing status
- And no "GitHub Spec Kit" banner is printed (D1)

## Scenario 2: JSON output is structured

<!-- id: scenario.cli.check.json -->

- Given `specify check --json`
- Then it emits a JSON array of `{ tool, found, path? }` records
- And the same information is present as in the plain output

## Scenario 3: Missing optional tools do not fail the command

<!-- id: scenario.cli.check.optional-missing -->

- Given an optional tool is missing
- When the user runs `specify check`
- Then the tool is reported missing but the command still exits 0 (check is informational)

## Scenario 4: A self-check tip does not hit the network

<!-- id: scenario.cli.check.no-network -->

- Given `specify check`
- Then any upgrade/self-check tip is computed without fetching the latest release (no network call in `check`)
