---
id: story.cli.version
kind: story
depends-on: [conventions]
---

# Story: Report the version

As an agent or developer,
I want `specify version` to report the binary version, plainly or as JSON,
So that I always know which CLI I am talking to and can parse it.

# Acceptance Criteria

## Scenario 1: Plain version

<!-- id: scenario.cli.version.plain -->

- Given a built `specify` binary
- When the user runs `specify version`
- Then it prints the version string and nothing else (no banner, D1)
- And it exits 0

## Scenario 2: JSON version is unconditional

<!-- id: scenario.cli.version.json -->

- Given a built `specify` binary
- When the user runs `specify version --json`
- Then it emits a JSON object with a `version` field
- And `--json` requires no other flag (diverges from upstream, which needs `--features`; D2)

## Scenario 3: The version is injected at build

<!-- id: scenario.cli.version.build-injected -->

- Given a release build
- When the binary reports its version
- Then the value is the release version injected at link time (goreleaser `-ldflags -X main.version=…`), not a hard-coded constant

## Scenario 4: A dev build reports a dev version

<!-- id: scenario.cli.version.dev -->

- Given a build with no injected version
- When the user runs `specify version`
- Then it reports a clearly-marked development version (e.g. `0.0.0-dev`)
