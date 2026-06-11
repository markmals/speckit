---
id: story.cli.upgrade
kind: story
depends-on: [story.cli.version]
---

# Story: Self-upgrade

As a developer,
I want `specify self upgrade` to update the binary in place,
So that I can stay current without a package manager, the same way the release was built (goreleaser).

# Acceptance Criteria

## Scenario 1: Reports current vs latest

<!-- id: scenario.cli.upgrade.reports -->

- Given `specify self upgrade`
- Then it resolves the latest release and reports the current and latest versions before changing anything

## Scenario 2: Already-latest is a clean no-op

<!-- id: scenario.cli.upgrade.noop-when-current -->

- Given the installed binary is already the latest release
- When the user runs `specify self upgrade`
- Then it reports "already up to date" and replaces nothing, exiting 0

## Scenario 3: Upgrade replaces the binary atomically

<!-- id: scenario.cli.upgrade.atomic-replace -->

- Given a newer release exists
- When the user runs `specify self upgrade`
- Then the new binary is downloaded, verified against its checksum, and swapped in atomically (no half-written binary on failure)

## Scenario 4: JSON output

<!-- id: scenario.cli.upgrade.json -->

- Given `specify self upgrade --json`
- Then it emits a structured `{ current, latest, action }` result (`action` ∈ upgraded | already-current | failed)

## Scenario 5: A network failure fails safe

<!-- id: scenario.cli.upgrade.network-fail-safe -->

- Given the latest release cannot be fetched
- When the user runs `specify self upgrade`
- Then it reports the failure and leaves the installed binary untouched
