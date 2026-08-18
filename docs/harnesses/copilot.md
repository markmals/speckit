# SpecKit with GitHub Copilot

GitHub Copilot reads its configuration from `.github/`. Running `specify init --integration copilot` wires SpecKit into Copilot by projecting everything — commands, orientation, skills, rules, and memory — under that one root. The wrinkle unique to this harness: each `/speckit.*` command is projected **twice** — once as a custom chat-mode under `.github/agents/` and once as a slash-prompt under `.github/prompts/` — so you can invoke a command either way in Copilot Chat.

## Initialize

```sh
specify init my-app --integration copilot
cd my-app
```

You get the nine `/speckit.*` commands (as both agents and prompts), a `.github/copilot-instructions.md` orientation file, the process skills and rules under `.github/`, a seed memory index, and the shared `.speckit/` runtime. `--here` initializes the current directory instead of creating `my-app`; `--force` merges into a non-empty directory; re-running `init` never clobbers accumulated memory (the seed `MEMORY.md` is written skip-if-exists).

## What init projects

| Artifact | Path |
| --- | --- |
| `/speckit.*` commands (chat-modes) | `.github/agents/speckit.<cmd>.agent.md` |
| `/speckit.*` commands (slash-prompts) | `.github/prompts/speckit.<cmd>.prompt.md` |
| Orientation file | `.github/copilot-instructions.md` |
| Process skills | `.github/skills/<skill>/SKILL.md` |
| Rules | `.github/rules/<rule>.md` |
| Memory | `.github/memory/MEMORY.md` (seed index) |
| Review subagents | none (Claude-only — see below) |

Alongside these, `init` writes the shared `.speckit/` runtime (the constitution, the spec/plan/tasks/checklist templates, `extensions.yml`, and — after a green `verify` — the per-spec lock under `.speckit/lock/<target>/<spec-id>.json`). The runtime is identical for every harness and holds no shell scripts; the command logic lives in the `specify` binary.

## The orientation file

Copilot's orientation file is `.github/copilot-instructions.md`. Unlike Claude's `CLAUDE.md`, it does **not** auto-import rules and memory — it uses a read-at-start directive that points the agent at `.github/memory/MEMORY.md`, and it references the rule files as prose (not `@import` lines). At the start of a task, Copilot reads the memory index and any topic files it points to, and follows the named rules.

```text
## Rules

Follow the conventions in `.github/rules/`: `code-quality.md`,
`commit-discipline.md`, and `spec-conventions.md`;
`enforcement-hierarchy.md` governs where a new convention lives.

## Project memory

At the start of a task, read the project memory index —
`.github/memory/MEMORY.md` — and any topic files it points to. It's
durable, agent-owned working knowledge about this repo (not spec truth; the engine
never reads it). Maintain it with the `managing-memory` skill.
```

## Driving the `/speckit.*` commands

Each command is projected two ways, so you have two ways to invoke it in Copilot Chat:

- **Slash-prompts** (`.github/prompts/speckit.<cmd>.prompt.md`) are invoked as slash commands — type `/speckit.specify`, `/speckit.plan`, and so on directly in chat.
- **Custom agents / chat-modes** (`.github/agents/speckit.<cmd>.agent.md`) are the same commands as selectable Copilot agents (chat-modes), with frontmatter `name: speckit.<cmd>`.

Both forms carry the identical, agent-neutral prompt body — the projection just exposes it through both of Copilot's invocation surfaces. Run the `specify` engine (`scan`, `verify`, `drift`, `parity`) in your terminal.

| Command | What it does |
| --- | --- |
| `/speckit.specify` | Author a new feature folder from a description (narrative, prioritized stories, models, view-models, errors). |
| `/speckit.clarify` | Resolve `[NEEDS CLARIFICATION]` markers by asking up to five targeted questions and writing the answers back into the specs. |
| `/speckit.analyze` | Read-only cross-artifact consistency/quality analysis of a feature; reports by severity, never edits. |
| `/speckit.checklist` | Generate a requirements-quality checklist ("unit tests for the spec"). |
| `/speckit.constitution` | Create or update the project constitution and propagate the change. |
| `/speckit.plan` | Produce a per-target technical plan for a feature. |
| `/speckit.tasks` | Build an ordered, story-prioritized task list for a target; each task maps to a spec ID. |
| `/speckit.implement` | Implement on a target: failing tests first, layered review, adversarial pass, then verify-and-lock. |
| `/speckit.taskstowork` | File a feature's task list as work items through the configured work provider (`specify work create`, one item per task). |

## Skills, rules, and memory

**Process skills** (`.github/skills/<skill>/SKILL.md`) — eight discipline skills the commands lean on: `test-driven-development` (RED/GREEN), `verification-before-completion`, `adversarial-review`, `systematic-debugging`, `implementing-a-spec`, `brainstorming-feature`, `writing-user-stories`, and `managing-memory`.

**Rules** (`.github/rules/<rule>.md`) — four always-relevant conventions, referenced from the orientation file: `code-quality`, `commit-discipline`, `spec-conventions`, and `enforcement-hierarchy` (the standard for deciding *where* a new convention should live).

**Memory** (`.github/memory/MEMORY.md`) — committed, repo-local, agent-owned working knowledge about this project: not spec truth, never verified or gated, and the engine never reads it. `MEMORY.md` is a one-line-per-topic index loaded each session; topic files are read on demand. Maintain it with the `managing-memory` skill. The test: a statement of required behavior belongs in the spec library; knowledge that helps an agent work here but isn't the behavior under test belongs in memory; anything already in code, git, or the specs belongs in neither. See [the agent-memory design](../design/agent-memory.md).

**No review subagents.** Copilot gets none. The `.github/agents/` directory here holds the **command chat-modes**, not dispatched reviewers — the review subagents (`spec-reviewer`, `test-gap-finder`, `drift-hunter`, `handoff-builder`, `visual-verifier`) are a Claude Code dispatch concept and are projected only for the `claude` integration.

## What's different for Copilot

- **One root.** Everything lands under `.github/` — Copilot's native config root — rather than a tool-specific top-level dir.
- **Dual command projection.** Every `/speckit.*` command exists as both a `.github/agents/*.agent.md` chat-mode and a `.github/prompts/*.prompt.md` slash-prompt. No other harness double-projects.
- **`.github/agents/` is for commands, not reviewers.** It holds the command chat-modes; the five review subagents are Claude-only.
- **Read-at-start orientation, not native import.** `.github/copilot-instructions.md` directs the agent to read `.github/memory/MEMORY.md` and references the rules as prose — unlike `CLAUDE.md`, which auto-imports both every session.

## Next

- [Offline engine usage](../usage/offline.md) — the engine alone: `scan` / `verify` / `drift` / `cover` / `parity` / `gate`
- [Working with GitHub](../usage/github.md) — the optional PR gate and the `github-projects` work provider
- [Project README](../../README.md) — overview and full command reference
- Other harnesses: [Claude Code](claude.md) · [Codex](codex.md) · [Generic (AGENTS.md)](generic.md)
