---
name: managing-memory
description: Use when you learn a durable, non-obvious fact about THIS repo that isn't already in the code, specs, or git history — a convention, a debugging pattern, why a workaround exists. Records it in the agent's repo-local memory/ store so it survives across sessions. Also use when reviewing or pruning that store.
---

# Managing memory

Agents have amnesia. Useful project knowledge — a debugging pattern, an API
convention that isn't itself a spec, why a workaround exists — evaporates between
sessions. The repo-local `memory/` store fixes that: it's committed markdown,
shared across sessions and humans, and loaded every session via its index.

This store is **agent-owned working knowledge, never spec truth.** The engine
ignores it entirely — `scan` / `verify` / `gate` never read it. Required behavior
belongs in the spec library (`features/`, `specs/`), which the engine joins, locks,
and gates. Memory is an aid, not a source of truth.

## Where it lives

Your memory dir depends on the agent (`specify init` projects it for you):

| Agent | Memory dir |
| --- | --- |
| Claude Code | `.claude/memory/` |
| Codex / generic | `.agents/memory/` |
| Copilot | `.github/memory/` |

`MEMORY.md` is the index — one line per topic file, loaded every session. Topic
files hold the detail and are read on demand.

## When to write a memory

Write one when you learn a **durable, non-obvious** fact about this repo that is
**not** already captured in code, specs, or git history.

- ✅ "the e2e suite needs `--no-sandbox` in CI or it hangs"
- ✅ "we standardized on the REST envelope `{data, error}`"
- ✅ "the offline determinism line: the engine never reads GitHub or agent dirs"
- ❌ required behavior → that's a scenario in the spec library
- ❌ something already obvious from the code, a spec, or `git log`
- ❌ facts that only matter to the task in front of you

The test: *"Is this required behavior?"* → spec. *"Knowledge that helps an agent
work here but isn't the behavior under test?"* → memory. *"Already in the code, git
history, or specs?"* → neither; don't duplicate it.

## The discipline

- **One topic per file; concise.** Granularity is your call — a topic, not one fact
  per file. Optional light frontmatter (`description:`) aids recall.
- **Add a one-line pointer to `MEMORY.md`.** Keep the index short — it's always in
  context: `- [Topic](topic.md) — one-line hook`.
- **Update, don't duplicate.** Edit the existing file rather than adding a second one.
  **Delete** a memory that turns out wrong.
- **Link related files** so a reader can follow the thread.
- **It's committed.** Memory diffs in PRs like any other doc; write it for the next
  human and the next session, not just for yourself.
