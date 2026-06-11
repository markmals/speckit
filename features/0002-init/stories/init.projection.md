---
id: story.init.projection
kind: story
depends-on: [story.init.basic]
---

# Story: Project the command set per agent

As a developer initializing a project for my agent,
I want the SpecKit command set projected into my agent's native format and location,
So that the same authored prompts work whether I use Claude Code, Codex, or Copilot — across CLI, GUI, and editor surfaces.

Reference manifests captured from the oracle live in `testdata/oracle-init/` (D14). The command prompts are shared data; the projection is per-agent location + wrapper.

# Acceptance Criteria

## Scenario 1: Claude projects to skills

<!-- id: scenario.init.projection.claude -->

- Given `specify init --integration claude`
- Then the command set is written to `.claude/skills/speckit-<cmd>/SKILL.md`
- And the orientation file is `CLAUDE.md`
- And an integration manifest records what was installed

## Scenario 2: Codex projects to skills + AGENTS.md

<!-- id: scenario.init.projection.codex -->

- Given `specify init --integration codex`
- Then the command set is written to `.agents/skills/speckit-<cmd>/SKILL.md`
- And the orientation file is `AGENTS.md` (the shared Codex/Copilot substrate, D4)

## Scenario 3: Copilot projects to agents + prompts

<!-- id: scenario.init.projection.copilot -->

- Given `specify init --integration copilot`
- Then the command set is written to `.github/agents/speckit.<cmd>.agent.md` and `.github/prompts/speckit.<cmd>.prompt.md`
- And the orientation file is `.github/copilot-instructions.md`

## Scenario 4: Generic projects to a configurable commands dir + AGENTS.md

<!-- id: scenario.init.projection.generic -->

- Given `specify init --integration generic --integration-options "--commands-dir <dir>"`
- Then the command set is written to `<dir>/speckit.<cmd>.md`
- And the orientation file is `AGENTS.md`

## Scenario 5: The prompt content is identical across agents

<!-- id: scenario.init.projection.shared-prompt-content -->

- Given the same command (e.g. `speckit.specify`)
- When it is projected for claude, codex, and copilot
- Then the prompt body is the same authored content, differing only in the agent-specific wrapper/location

## Scenario 6: The runtime base diverges from upstream

<!-- id: scenario.init.projection.fork-divergence -->

- Given any `specify init`
- Then the runtime directory is `.speckit/`, never `.specify/` (D6)
- And no `scripts/bash/*.sh` are written — command logic lives in the `specify` binary, with only git-hook trampolines on disk (D2)
- And no workflow engine or `workflows/` directory is installed (deferred)
- And no "GitHub Spec Kit" banner appears in any output (D1)

## Scenario 7: Adding an agent surface is one adapter

<!-- id: scenario.init.projection.one-adapter -->

- Given a new agent or surface to support
- When it is added
- Then it is implemented as a single projection adapter (one Go interface), with the shared prompts unchanged (D4)
