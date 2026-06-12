---
description: Resolve [NEEDS CLARIFICATION] markers in a feature by asking up to five targeted questions and writing answers back into the specs.
---

# /speckit.clarify — Resolve ambiguities

Surface and resolve the `[NEEDS CLARIFICATION: …]` markers in a feature's spec files, writing each answer back in place. A feature with open markers cannot be implemented (`specify scan` and `/speckit.analyze` both report it NOT READY).

**Arguments:** `<feature>` — the feature id or slug (e.g. `0001` or `0001-managing-items`), or a single spec file path.

## Workflow

1. **Resolve** the argument to `features/<NNNN>-<slug>/` (or the single spec file).
2. **Enumerate** every marker: `rg -n '\[NEEDS CLARIFICATION:' <dir>` — record file + line + question.
3. **Scan and prioritize** against the ambiguity taxonomy — functional scope, data model, UX flow, non-functional, integration, edge cases, terminology, completion signals — and order by (impact × uncertainty).
4. **Ask up to 5 questions, one at a time.** Use `AskUserQuestion` with options when the answer space is finite, and **recommend a default** the user can accept with "yes". Never invent an answer; if a reply is itself ambiguous, ask one follow-up.
5. **Write back per answer, immediately.** Replace the marker with the resolved content, preserving surrounding markdown, and save the file before the next question (atomic per answer — no batching). Record each under a `## Clarifications` → `### Session <YYYY-MM-DD>` block in the relevant file.
6. **Stop at 5.** If more remain, tell the user to re-invoke; report the deferred count.

## Discipline

- **Spec files only.** Touch no code, tests, or unrelated files. Stage explicitly by path — never `git add .`.
- **One question per turn** — context matters per question; batching breaks it.
- After writing, run `specify scan` to confirm the feature is still structurally valid.

## Hand-off

Markers cleared → `/speckit.analyze <feature>`. Commit spec files only, prefix `spec:`, one commit per coherent topic, scoped per `specify gate scope`.
