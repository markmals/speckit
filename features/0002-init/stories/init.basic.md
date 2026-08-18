---
id: story.init.basic
kind: story
depends-on: [conventions]
---

# Story: Initialize a project for an agent

As a developer starting a SpecKit project,
I want `specify init` to set up a working project projected for my agent,
So that I can begin authoring specs immediately, with the runtime binary present and no script layer to maintain.

# Acceptance Criteria

## Scenario 1: Fresh init produces a working, agent-projected tree

<!-- id: scenario.init.basic.fresh -->

- Given an empty directory
- When the user runs `specify init --here` and selects an agent (claude, codex, or copilot)
- Then the chosen agent's command prompts are written in that agent's native format
- And the spec conventions and process prompts are present
- And every generated command prompt invokes `specify`, with zero runtime scripts beyond git-hook trampolines

## Scenario 2: The same specs project per agent

<!-- id: scenario.init.basic.per-agent-projection -->

- Given the same project content
- When `specify init` is run for claude, for codex, and for copilot
- Then each produces the expected tree for that agent family's shared config format
- And the golden-tree fixture for each agent matches

## Scenario 3: Init is safe in a non-empty directory

<!-- id: scenario.init.basic.non-empty-guard -->

- Given a directory that already contains files
- When the user runs `specify init --here` without `--force`
- Then the command refuses rather than overwriting
- And with `--force` it proceeds, leaving unrelated files untouched

## Scenario 4: Init works on all three host OSes

<!-- id: scenario.init.basic.cross-os -->

- Given a supported host (Linux, macOS, or Windows)
- When the user runs `specify init`
- Then the resulting project is identical modulo path separators and the git-hook trampoline variant (`.sh` vs `.cmd`, D10)
