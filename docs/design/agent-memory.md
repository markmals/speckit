# Design — agent memory (per-agent `memory/`)

**Status:** proposal / direction-setting. Bring Claude Code's file-based memory
pattern into SpecKit as a **repo-local, committed** store, projected into each
agent's own directory (the same way SpecKit already projects skills), so the agent
accumulates durable working knowledge about *this* repo that survives context
compaction and is shared across sessions and humans.

## Thesis

Agents have amnesia; useful project knowledge — a debugging pattern, why a
workaround exists, an API convention that isn't itself a spec — evaporates between
sessions. Claude Code solves this with a *per-user* memory dir. SpecKit makes it
**repo-local and committed**, so the knowledge is the project's, not one user's.

This is the concrete form of the GitHub-native decision *"durable agent memory
stays as repo markdown, not a pinned issue."*

## The store — per-agent, like skills

Memory lands in the agent's own repo-local directory, exactly parallel to how
SpecKit projects skills today:

| Agent | Skills (existing) | Memory |
| --- | --- | --- |
| Claude Code | `.claude/skills/` | `.claude/memory/` |
| Codex / generic | `.agents/skills/` | `.agents/memory/` |
| Copilot | `.github/skills/` | `.github/memory/` |

"Agent-agnostic" means the **mechanism** is identical for every agent and SpecKit
projects it to the right place — *not* that there's one shared cross-agent store.
A project usually has one primary agent, so that agent's `memory/` *is* the
project's memory (multi-agent is open decision 3).

The structure is the same regardless of root — Claude Code shown:

```
<repo>/.claude/memory/
├── MEMORY.md            # concise index, loaded into every session
├── debugging.md         # detailed notes on debugging patterns
├── api-conventions.md   # API design decisions
└── …                    # any topic file the agent creates
```

- **`MEMORY.md` is the index**, loaded every session: one line per topic file
  (`- [API conventions](api-conventions.md) — REST envelope + error shape`). Keep
  it short; it's always in context. Topic files are read on demand when relevant.
- **Topic files are agent-authored** — one coherent topic each, light optional
  frontmatter (`description` for recall). The agent creates/updates/deletes them as
  it learns; granularity is the agent's call (a topic, not one fact per file).
- **Committed.** Unlike Claude Code's per-user store, this lives in the repo and is
  version-controlled — humans and future sessions see the same memory, and it diffs
  in PRs like any other doc.

## Loading (per-agent projection by `init`)

`specify init` wires the chosen agent's instruction file to load the index from
*that agent's* `memory/`, using a native mechanism where one exists and a directive
where it doesn't:

| Agent | Mechanism |
| --- | --- |
| Claude Code | `CLAUDE.md` gets `@.claude/memory/MEMORY.md` — a native import, auto-loaded every session |
| Codex / generic | `AGENTS.md` gets a **Project memory** section with a read-at-start directive → `.agents/memory/MEMORY.md` (no universal import) |
| Copilot | `.github/copilot-instructions.md` — the same directive → `.github/memory/MEMORY.md` |

The index is small, so loading it every session is cheap; topic files are pulled
only when a session's work touches them.

## What belongs in memory vs the spec library

The line matters — memory must not become a second, unverified source of truth.

| | Spec library (`features/`, `specs/`) | Agent memory (`<agent>/memory/`) |
| --- | --- | --- |
| What | required **behavior** | working **knowledge** about the repo |
| Verified? | yes — the engine joins, locks, gates it | no — never verified or gated |
| Examples | scenarios, models, acceptance criteria | "the flaky test needs `--no-sandbox` in CI", "we standardized on the REST envelope `{data,error}`", why a workaround exists |
| Source of truth? | **yes** | no — an aid, not truth |

The test: *"Is this a statement of required behavior?"* → spec. *"Knowledge that
helps an agent work here but isn't the behavior under test?"* → memory. *"Already
in the code, git history, or specs?"* → neither; don't duplicate it.

**The agent dirs (`.claude/`, `.agents/`, `.github/`) are agent-owned;
`.speckit/` is engine-owned.** Memory is freely edited by agents and is **not**
covered by `gate generated` (which protects engine I/O like locks). The engine
**ignores memory entirely** — `scan`/`verify`/`gate` never read it.

## The discipline (a projected skill)

SpecKit ships a `managing-memory` skill (projected like the others) teaching the
discipline — adapted from how Claude Code maintains memory:

- Write a memory when you learn a **durable, non-obvious** fact about this repo not
  already captured in code/specs/git.
- One topic per file; concise. Add a one-line pointer to `MEMORY.md`.
- Keep `MEMORY.md` short — it's loaded every session.
- Update an existing file rather than duplicating; delete memories that turn out
  wrong.
- Link related files. Don't store what the repo already records — if asked to
  remember something derivable, capture only what was non-obvious about it.

## What `specify` does

- `init` seeds the chosen agent's `<root>/memory/MEMORY.md` (a starter index) and
  wires the loading above; ships the `managing-memory` skill.
- The **engine ignores memory entirely** — it's agent context, not spec state.
  (This keeps the integrity core clean.)
- SpecKit **dogfoods it**: this repo is developed with Claude Code, so it gets a
  `.claude/memory/` (we can migrate the project-relevant notes there when we build).

## Open decisions

1. **Frontmatter** — require light frontmatter (`description`) on topic files for
   better recall, or leave them free-form markdown? Lean: optional `description`,
   nothing mandatory.
2. **Personal vs shared** — everything committed for v1. A gitignored
   `<root>/memory/local/` for personal scratch is a possible later addition; YAGNI
   now.
3. **Multi-agent projects** — if a repo uses more than one agent, does each keep its
   own `memory/` (default, simplest), or is there one canonical store the others
   reference/symlink? Lean: per-agent for v1; revisit if real demand appears.
4. **`AGENTS.md` include** — if/when a cross-agent include syntax becomes common, use
   it instead of the read-at-start directive for the non-Claude agents.
