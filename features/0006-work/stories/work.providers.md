---
id: story.work.providers
kind: story
depends-on: [domain.work-item]
---

# Story: The provider is chosen at adoption time

As a team adopting SpecKit,
I want to pick a work provider in config — or none —
So that coordination fits how we already track work, and the engine never depends on the choice.

# Acceptance Criteria

## Scenario 1: Markdown is the default

<!-- id: scenario.work.providers.markdown-is-default -->

- Given a config with no `work` block
- When a work verb runs
- Then the markdown provider is used — an absent block is not an error

## Scenario 2: None is quiet

<!-- id: scenario.work.providers.none-is-quiet -->

- Given provider `none`
- When any work verb runs
- Then it prints one line saying no work provider is configured
- And exits 0

## Scenario 3: Beads requires the bd CLI

<!-- id: scenario.work.providers.beads-requires-bd -->

- Given the beads provider and no `bd` CLI on the path
- When a work verb runs
- Then it fails with a message naming the install step

## Scenario 4: Beads maps onto bd's native semantics

<!-- id: scenario.work.providers.beads-maps-native -->

- Given the beads provider
- Then it maps onto `bd`'s own ready predicate, typed dependencies, and compare-and-set claim
- And does not reimplement them

## Scenario 5: GitHub Projects drives a v2 board

<!-- id: scenario.work.providers.github-projects-board -->

- Given the github-projects provider with a configured owner and board number
- Then the five verbs drive a Projects v2 board
- And the status field and column names are overridable

## Scenario 6: The engine never requires a provider

<!-- id: scenario.work.providers.engine-never-requires-a-provider -->

- Given `scan`, `verify`, `drift`, `cover`, `parity`, and `gate`
- Then they neither require, invoke, nor import any provider
- And an absent `work` block is never an error for them

## Scenario 7: An unknown provider is rejected

<!-- id: scenario.work.providers.unknown-provider-rejected -->

- Given a config naming a provider id outside `markdown` | `beads` | `github-projects` | `none`
- When the user runs `specify scan`
- Then the unknown provider id is reported

## Scenario 8: The firewall is structural

<!-- id: scenario.work.providers.import-firewall -->

- Given the engine packages (`internal/engine`, `internal/specmodel`, `internal/reports`, `internal/config`)
- Then their transitive imports contain no work provider and no GitHub client
- And so every engine command runs with no network and no credentials
