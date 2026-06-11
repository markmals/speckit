---
id: story.extension.install
kind: story
depends-on: [story.init.basic]
---

# Story: Install and manage platform extensions

As a developer choosing which platforms to ship,
I want `--platforms` and `specify extension add/remove` to install bundled packs additively,
So that the curated distro is just the first-party catalog — additive, reversible, and versioned (D3) — with no superset to prune.

# Acceptance Criteria

## Scenario 1: `--platforms` scaffolds buildable apps with wired adapters

<!-- id: scenario.extension.install.platforms-sugar -->

- Given an empty directory
- When the user runs `specify init --platforms web,apple --backend convex`
- Then the `platform-web` and `platform-apple` packs and `backend-convex` are installed from the vendored catalog
- And two buildable app scaffolds exist, each with a wired verify-adapter manifest

## Scenario 2: The first-party catalog resolves offline

<!-- id: scenario.extension.install.offline -->

- Given no network connection
- When the user runs `specify init --platforms web`
- Then the bundled first-party catalog resolves and the pack installs without any network access

## Scenario 3: Add then remove round-trips cleanly

<!-- id: scenario.extension.install.round-trip -->

- Given a project with a pack installed
- When the user runs `specify extension remove <pack>`
- Then any command versions the pack had overridden are restored to their prior state
- And re-running `specify extension add <pack>` returns the project to the installed state

## Scenario 4: A community extension is not promised to install (D6)

<!-- id: scenario.extension.install.community-reference-only -->

- Given an extension from the snapshotted upstream community catalog that hardcodes `.specify/scripts/bash/*.sh` paths
- When the user attempts to install it
- Then the tool does not silently pretend success: it reports that community extensions are reference-only, because the fork has no `.specify/` compat layer
