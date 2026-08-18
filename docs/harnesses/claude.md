# SpecKit with Claude Code

Claude Code is Anthropic's agentic coding harness. Running `specify init --integration claude` wires SpecKit into it: the `/speckit.*` commands become user-invocable skills, the process-discipline skills land in `.claude/skills/`, the rules and memory index in `.claude/`, and `CLAUDE.md` orients the agent. This is the richest of the four integrations — Claude is the only harness that also gets the five review subagents, and the only one whose orientation file loads rules and memory through Claude Code's native `@import` rather than a read-at-start directive.

## Initialize

```sh
specify init my-app --integration claude
cd my-app
```

You get the `/speckit.*` command skills, the eight process-discipline skills, the four rules, a seed memory index, the five review subagents, a `CLAUDE.md` orientation file, and the shared `.speckit/` runtime. Use `--here` to set up the current directory instead of a new one, and `--force` to merge into a non-empty directory (re-running `init` never clobbers accumulated memory — the seed `MEMORY.md` is written skip-if-exists).

## What init projects

| Artifact | Path |
| --- | --- |
| `/speckit.*` commands | `.claude/skills/speckit-<cmd>/SKILL.md` |
| Orientation file | `CLAUDE.md` |
| Process-discipline skills | `.claude/skills/<skill>/SKILL.md` |
| Rules | `.claude/rules/<rule>.md` |
| Memory | `.claude/memory/MEMORY.md` (seed index) |
| Review subagents (Claude only) | `.claude/agents/<name>.md` |

The shared `.speckit/` runtime is written identically for every agent — the constitution (`.speckit/memory/constitution.md`), the spec/plan/tasks/checklist templates under `.speckit/templates/`, and `.speckit/extensions.yml`; the per-spec locks appear under `.speckit/lock/<target>/<spec-id>.json` after a green `verify`. No shell scripts: the command logic lives in the binary.

## The orientation file

The orientation file is `CLAUDE.md`. It is where Claude Code differs from the other harnesses: instead of a "read this at the start of a session" directive, it uses Claude Code's **native `@import`** — the file literally contains `@`-prefixed paths that Claude Code auto-loads every session. The three always-loaded rules and the memory index are imported directly:

```text
## Rules

These always-loaded conventions live in `.claude/rules/`:

@.claude/rules/code-quality.md
@.claude/rules/commit-discipline.md
@.claude/rules/spec-conventions.md

`.claude/rules/enforcement-hierarchy.md` is the standard for deciding
where a new convention lives — read it when you add one.

## Memory

Durable, non-obvious project knowledge lives in `.claude/memory/`; the
index is auto-loaded every session:

@.claude/memory/MEMORY.md
```

So `code-quality`, `commit-discipline`, `spec-conventions`, and the memory index are loaded on every session with no read-at-start instruction needed. `enforcement-hierarchy.md` is the one rule that is *not* auto-imported — it's referenced as the standard to read when you add a new convention, since you reach for it only at that moment.

## Driving the /speckit.* commands

The commands are **user-invocable skills** — the `/speckit.*` command family. On disk the skill directory and its frontmatter `name` use a hyphen (`speckit-<cmd>`) and carry `user-invocable: true`; the product-facing command family is the dotted `/speckit.*` set. Invoke them inside Claude Code as `/speckit.specify`, `/speckit.plan`, and so on; run `specify …` (the engine) in your terminal.

| Command | What it does |
| --- | --- |
| `/speckit.specify` | Author a new feature folder from a description (narrative, prioritized stories, models, view-models, errors). |
| `/speckit.clarify` | Resolve `[NEEDS CLARIFICATION]` markers by asking up to five targeted questions and writing the answers back into the specs. |
| `/speckit.analyze` | Read-only cross-artifact consistency and quality analysis of a feature; reports by severity, never edits. |
| `/speckit.checklist` | Generate a requirements-quality checklist ("unit tests for the spec"). |
| `/speckit.constitution` | Create or update the project constitution and propagate the change. |
| `/speckit.plan` | Produce a per-target technical plan for a feature. |
| `/speckit.tasks` | Produce an ordered, story-prioritized task list for a target; each task maps to a spec ID. |
| `/speckit.implement` | Implement on a target: failing tests first, layered review, an adversarial pass, then verify-and-lock. |
| `/speckit.taskstowork` | File a feature's task list as work items through the configured work provider (`specify work create`, one item per task). |

## Skills, rules, and memory

Beyond the commands, `init` projects eight **process-discipline skills** into `.claude/skills/`: `test-driven-development` (RED/GREEN), `verification-before-completion`, `adversarial-review`, `systematic-debugging`, `implementing-a-spec`, `brainstorming-feature`, `writing-user-stories`, and `managing-memory`. These encode how Claude should work, not what to build.

The four **rules** land in `.claude/rules/`: `code-quality`, `commit-discipline`, `spec-conventions`, and `enforcement-hierarchy`. The first three are auto-imported by `CLAUDE.md`; `enforcement-hierarchy` is the standard for deciding *where* a new convention should live.

The **memory** store is `.claude/memory/`, seeded with a `MEMORY.md` index — one line per topic, auto-loaded each session through `@import`, with topic files read on demand. It is committed, repo-local, agent-owned working knowledge: not spec truth, never verified or gated, and the engine never reads it. Maintain it with the `managing-memory` skill. The test: a statement of required behavior belongs in the spec library; knowledge that helps an agent work here but is not the behavior under test belongs in memory; anything already in code, git, or the specs belongs in neither.

### Review subagents

Claude is the **only** harness that gets review subagents. The init projects five of them into `.claude/agents/`, each dispatched on demand:

| Subagent | What it does |
| --- | --- |
| `spec-reviewer` | Audits a spec before it lands — confirms it passes `specify scan`, then checks Gherkin scenarios for stable sub-IDs, unambiguous language, `[NEEDS CLARIFICATION]` markers, target-neutrality, and reverse-pointer health. Read-only. |
| `test-gap-finder` | Finds Gherkin scenarios with no bound, passing test on a given target; runs `specify verify` and reports uncovered or failing scenarios. Read-only. |
| `drift-hunter` | Audits cross-target spec/impl drift via `specify drift`/`verify`/`parity` and returns a prioritized punch list. Read-only. |
| `handoff-builder` | Generates or updates `HANDOFF.md` from current branch state so the next session can pick up with full context. |
| `visual-verifier` | Walks a feature through its Gherkin scenarios on a GUI target (web, iOS simulator, Android emulator), screenshots each state, and reports rendering mismatches. |

## What's different for Claude Code

- **Native `@import`, not a directive.** `CLAUDE.md` auto-loads the three core rules and the memory index every session via `@.claude/...md` lines. The other harnesses (`AGENTS.md` for codex/generic, `.github/copilot-instructions.md` for copilot) carry a read-at-start directive and reference rules as prose instead.
- **Review subagents are Claude-only.** The five-agent review pack in `.claude/agents/` is a Claude-dispatch concept; codex, generic, and copilot get none. (Copilot's `.github/agents/` holds the *command* chat-modes, not dispatched reviewers.)
- **Skills root is `.claude/`.** Commands, process skills, rules, and memory all land under `.claude/`, where the codex/generic projection uses `.agents/` and copilot uses `.github/`.

SpecKit itself is developed with Claude Code and dogfoods its own `.claude/memory/`.

## Next

- [Offline engine usage](../usage/offline.md) — the engine alone: `scan` / `verify` / `drift` / `cover` / `parity` / `gate`
- [Working with GitHub](../usage/github.md) — the optional PR gate and the `github-projects` work provider
- [Project README](../../README.md) — overview and full command reference
- Other harnesses: [Codex](codex.md) · [Generic (AGENTS.md)](generic.md) · [GitHub Copilot](copilot.md)
