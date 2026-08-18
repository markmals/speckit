# SpecKit with a generic agent (AGENTS.md)

This is the portable fallback: any coding agent that reads `AGENTS.md` and the
`.agents/` convention — and isn't Claude Code, Codex, or Copilot — wires into
SpecKit with `specify init --integration generic`. The projection is **byte-for-byte
identical to [Codex](codex.md)** (both run the same `agentsAdapter`); the only
difference is the integration id you pass. If your agent reads `AGENTS.md`, this
is how you connect it.

## Initialize

```sh
specify init my-app --integration generic
cd my-app
```

You get the `/speckit.*` commands as a skill set under `.agents/skills/`, an
`AGENTS.md` orientation file, the eight process skills, the four rules, a seed
memory index, and the shared `.speckit/` runtime. `--integration` is required
(no default). Use `--here` to set up the current directory, and `--force` to
merge into a non-empty one. Re-running `init` never clobbers accumulated
memory — the seed `MEMORY.md` is written skip-if-exists.

## What init projects

| Artifact | Path |
| --- | --- |
| `/speckit.*` commands | `.agents/skills/speckit-<cmd>/SKILL.md` |
| Orientation file | `AGENTS.md` |
| Process skills | `.agents/skills/<skill>/SKILL.md` |
| Rules | `.agents/rules/<rule>.md` |
| Memory | `.agents/memory/MEMORY.md` (seed index) |
| Review subagents | none (Claude-only) |

Everything lands under `.agents/` — the same convention Codex uses. The
`.speckit/` runtime (the constitution, the spec/plan/tasks/checklist templates,
`extensions.yml`, and — after a green `verify` — the per-spec locks under
`.speckit/lock/`) is written identically for every agent and shared across them.

## The orientation file

The orientation file is `AGENTS.md`. Unlike Claude's `CLAUDE.md`, it has no
native `@import` — your agent loads the rules and memory through a **read-at-start
directive** written as prose. Rules are referenced the same way: a "Rules"
section names the four files, but they are not auto-imported.

```text
# SpecKit

This project uses SpecKit. The `/speckit.*` commands live in
`.agents/skills/` and the runtime state in `.speckit/`.
Run `specify` for the engine (scan / verify / drift / parity).

## Rules

Follow the conventions in `.agents/rules/`: `code-quality.md`,
`commit-discipline.md`, and `spec-conventions.md` apply to
every change; `enforcement-hierarchy.md` is the standard for where a new
convention lives.

## Project memory

At the start of a session, read the project memory index —
`.agents/memory/MEMORY.md` — and any topic files it points to. ...
```

## Driving the /speckit.* commands

The commands are projected as your agent's skill set under `.agents/skills/`, and
`AGENTS.md` orients the agent to them. Drive them however that agent invokes its
skills or commands; run `specify …` in your terminal for the engine checks.

| Command | What it does |
| --- | --- |
| `/speckit.specify` | Author a new feature folder from a description (narrative, prioritized stories, models, view-models, errors). |
| `/speckit.clarify` | Resolve `[NEEDS CLARIFICATION]` markers by asking up to five targeted questions and writing answers back into the specs. |
| `/speckit.analyze` | Read-only cross-artifact consistency/quality analysis of a feature; reports by severity, never edits. |
| `/speckit.checklist` | Generate a requirements-quality checklist ("unit tests for the spec"). |
| `/speckit.constitution` | Create/update the project constitution and propagate the change. |
| `/speckit.plan` | Per-target technical plan for a feature. |
| `/speckit.tasks` | Ordered, story-prioritized task list for a target; each task maps to a spec ID. |
| `/speckit.implement` | Implement on a target: failing tests first, layered review, adversarial pass, then verify-and-lock. |
| `/speckit.taskstowork` | File a feature's task list as work items through the configured work provider (`specify work create`, one item per task). |

## Skills, rules, and memory

**Process skills (8)** land in `.agents/skills/` alongside the commands:
`test-driven-development` (RED/GREEN), `verification-before-completion`,
`adversarial-review`, `systematic-debugging`, `implementing-a-spec`,
`brainstorming-feature`, `writing-user-stories`, and `managing-memory`.

**Rules (4)** land in `.agents/rules/` and are referenced from `AGENTS.md`:
`code-quality`, `commit-discipline`, `spec-conventions`, and
`enforcement-hierarchy` — the last is the standard for deciding *where* a new
convention should live.

**Memory** is `.agents/memory/MEMORY.md`: committed, repo-local, agent-owned
working knowledge about this repo — not spec truth, never verified or gated. It's
a one-line-per-topic index loaded each session; topic files are read on demand.
Maintain it with the `managing-memory` skill. The engine never reads it.

There are **no review subagents** for a generic agent. The five-reviewer pack
(`spec-reviewer`, `test-gap-finder`, `drift-hunter`, `handoff-builder`,
`visual-verifier`) is a Claude-dispatch concept — it relies on Claude Code's
ability to dispatch subagents — so only the `claude` integration gets it.

## What's different for a generic agent

- **Identical to Codex on disk.** Same `agentsAdapter`, same `.agents/` tree,
  same `AGENTS.md`. The integration id is the only thing that changes. See
  [codex.md](codex.md).
- **Read-at-start, not auto-import.** `AGENTS.md` tells the agent to read the
  memory index and follow the rules; nothing is pulled in via native `@import`
  the way Claude's `CLAUDE.md` does it.
- **No review subagents.** The review pack is Claude-only.
- **The portable wiring.** This integration exists for any AGENTS.md-aware agent
  that isn't one of the named three — if your tool reads `AGENTS.md`, it works
  here.

## Next

- [Offline engine usage](../usage/offline.md) — the engine alone: `scan` / `verify` / `drift` / `cover` / `parity` / `gate`
- [Working with GitHub](../usage/github.md) — the optional PR gate and the `github-projects` work provider
- [Project README](../../README.md) — overview and full command reference
- Other harnesses: [Claude Code](claude.md) · [Codex](codex.md) · [GitHub Copilot](copilot.md)
