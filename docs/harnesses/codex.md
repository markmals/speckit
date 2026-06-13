# SpecKit with Codex

The OpenAI Codex CLI is an agent that reads `AGENTS.md` for project orientation and drives a set of skills. `specify init --integration codex` wires SpecKit into it: the `/speckit.*` authoring/implementation commands, the process-discipline skills, the rules, and the seed memory index, all under `.agents/`, with `AGENTS.md` as the entry point. The one thing to know up front: the `codex` and `generic` integrations produce **identical on-disk output** — they share the same adapter — so this guide and [generic.md](generic.md) describe the same projection. The only Claude-specific extra (the review subagents) is absent here.

## Initialize

```sh
specify init my-app --integration codex
cd my-app
```

`init` needs a project name (or `--here` for the current directory) and `--integration codex` — the integration id is required, there is no default. `--force` merges into a non-empty directory. Re-running `init` never clobbers accumulated memory: the seed `MEMORY.md` is written skip-if-exists.

What comes back: the `/speckit.*` commands wired as skills under `.agents/skills/`, the process skills and rules under `.agents/`, a seed memory index, an `AGENTS.md` orientation file, and the shared `.speckit/` runtime.

## What init projects

| Artifact | Path |
| --- | --- |
| `/speckit.*` commands | `.agents/skills/speckit-<cmd>/SKILL.md` |
| Orientation file | `AGENTS.md` |
| Process skills | `.agents/skills/<skill>/SKILL.md` |
| Review subagents | none (Claude-only) |
| Rules | `.agents/rules/<rule>.md` |
| Memory | `.agents/memory/MEMORY.md` (seed index) |

The `.speckit/` runtime is written identically for every agent: the constitution (`.speckit/memory/constitution.md`), the spec/plan/tasks/checklist templates under `.speckit/templates/`, and `extensions.yml`. After a green `verify`, the per-spec locks appear under `.speckit/lock/<target>/<spec-id>.json`. There are no shell scripts — the command logic lives in the `specify` binary.

## The orientation file

The orientation file for Codex is `AGENTS.md` at the project root. Unlike Claude's `CLAUDE.md` (which uses native `@import` lines auto-loaded every session), `AGENTS.md` uses a **read-at-start directive** for memory and references the rules as **prose** — nothing is auto-imported. Its "Project memory" section instructs the agent to read the memory index at the start of a session, and its "Rules" section names the four rule files to follow:

```text
## Rules

Follow the conventions in `.agents/rules/`: `code-quality.md`,
`commit-discipline.md`, and `spec-conventions.md` apply to
every change; `enforcement-hierarchy.md` is the standard for where a new
convention lives.

## Project memory

At the start of a session, read the project memory index —
`.agents/memory/MEMORY.md` — and any topic files it points to. It's
durable, agent-owned working knowledge about this repo (not spec truth; the engine
never reads it). Maintain it with the `managing-memory` skill.
```

## Driving the /speckit.* commands

The nine commands are projected as Codex's skill set under `.agents/skills/speckit-<cmd>/SKILL.md` (each with `name: "speckit-<cmd>"`, a `description`, and `user-invocable: true` in its frontmatter — the same `skillDoc` as Claude). `AGENTS.md` orients Codex to them; you drive them however Codex invokes its skills/commands. Run `specify …` for the engine in your terminal.

| Command | What it does |
| --- | --- |
| `/speckit.specify` | Author a new feature folder from a description — narrative, prioritized stories, models, view-models, errors. |
| `/speckit.clarify` | Resolve `[NEEDS CLARIFICATION]` markers by asking up to five targeted questions and writing the answers back into the specs. |
| `/speckit.analyze` | Read-only cross-artifact consistency/quality analysis of a feature; reports by severity, never edits. |
| `/speckit.checklist` | Generate a requirements-quality checklist — "unit tests for the spec." |
| `/speckit.constitution` | Create or update the project constitution and propagate the change. |
| `/speckit.plan` | Produce the per-target technical plan for a feature. |
| `/speckit.tasks` | Produce an ordered, story-prioritized task list for a target; each task maps to a spec ID. |
| `/speckit.implement` | Implement on a target: failing tests first, layered review, an adversarial pass, then verify-and-lock. |
| `/speckit.taskstoissues` | File a feature's task list as GitHub issues on the repo matching the git remote. (The one command that touches GitHub.) |

## Skills, rules, and memory

**Process skills** (8, under `.agents/skills/`): `test-driven-development` (RED/GREEN), `verification-before-completion`, `adversarial-review`, `systematic-debugging`, `implementing-a-spec`, `brainstorming-feature`, `writing-user-stories`, and `managing-memory`. These encode the working discipline the `/speckit.*` commands lean on.

**Rules** (4, under `.agents/rules/`, referenced from `AGENTS.md`): `code-quality`, `commit-discipline`, and `spec-conventions` apply to every change; `enforcement-hierarchy` is the standard for deciding where a new convention should live.

**Memory** (`.agents/memory/`): a committed, repo-local, agent-owned store of working knowledge about this repo — not spec truth, never verified or gated. `MEMORY.md` is a one-line-per-topic index that the read-at-start directive loads each session; topic files are read on demand. Maintain it with the `managing-memory` skill. The engine never reads it.

There are **no review subagents** under the Codex projection: the review pack (`spec-reviewer`, `test-gap-finder`, `drift-hunter`, `handoff-builder`, `visual-verifier`) is a Claude-dispatch concept and projects only into `.claude/agents/`, so Codex gets none.

## What's different for Codex

- **Identical to `generic`.** The `codex` and `generic` integrations share the `agentsAdapter` and write byte-for-byte the same files under `.agents/` plus `AGENTS.md`. If you switch between them, nothing on disk changes. See [generic.md](generic.md).
- **`AGENTS.md`, not `CLAUDE.md`.** Orientation lives in `AGENTS.md`, and memory loads via a read-at-start directive rather than a native auto-import; the rules are referenced as prose.
- **No review subagents.** Unlike the Claude projection, there is no `.agents/agents/` directory — the review pack is Claude-only.

## Next

- [Offline engine usage](../usage/offline.md) — the engine alone: `scan` / `verify` / `drift` / `cover` / `parity` / `gate`
- [Working with GitHub](../usage/github.md) — the optional PR gate, Issues, Projects, deploys, secrets
- [Project README](../../README.md) — overview and full command reference
- Other harnesses: [Claude Code](claude.md) · [Generic (AGENTS.md)](generic.md) · [GitHub Copilot](copilot.md)
