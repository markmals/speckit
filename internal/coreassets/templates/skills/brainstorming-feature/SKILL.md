---
name: brainstorming-feature
description: Use before starting any new feature or substantial change — the discipline behind /speckit.specify. Walks the user from narrative → prioritized stories → models → view-models → flows → errors, populating a feature folder. The feature folder IS the spec; there is no separate plan document at this stage.
---

# Brainstorming a Feature

Shape an idea into a populated feature folder. By the end, `features/<NNNN>-<slug>/` holds a NARRATIVE plus enough stories, models, view-models, flows, and errors to drive `/speckit.plan` and `/speckit.implement`. Structure follows `specs/CONVENTIONS.md` exactly (it's mechanized in `internal/specmodel`, so the engine will reject a malformed folder).

**Default workspace:** the user's current branch. No worktree/branch ceremony unless the user asks.

## When to use

- Adding a new feature, or substantially changing an existing one (more than a bug fix).
- The user has an idea but isn't sure what shape the spec should take.

**Do NOT use for:** bug fixes (→ `systematic-debugging`); a one-scenario tweak to an existing feature (edit the file directly); cross-cutting architecture (→ `specs/ARCHITECTURE.md`, not a feature).

## The hard gate

Do **not** invoke `/speckit.implement` or write implementation code until the feature folder has a substantive `NARRATIVE.md`, one or more `stories/<id>.md` with Gherkin scenarios, and **explicit user approval** of the spec content. Skipping this produces specs that look complete but hide assumptions; code built on them gets reworked.

## Process

```
1. Scope check     — one feature, or several?
2. Explore context — read CONVENTIONS, ARCHITECTURE, related features
3. Question round  — one at a time, multiple-choice when finite
4. Approach round  — 2-3 approaches with tradeoffs; recommend one
5. Author          — NARRATIVE first, then stories (prioritized), then models/VMs/flows/errors
6. Self-review     — placeholders, contradictions, scope creep, [NEEDS CLARIFICATION] count
7. Validate        — specify scan (structural), then the user-review gate
8. Hand-off        — /speckit.clarify or /speckit.analyze
```

### 1. Scope check

If the prompt names multiple unrelated capabilities ("items plus calendar plus messaging"), stop and propose a decomposition: what's the independent first slice? Each feature must produce a working, testable capability on its own.

### 2. Explore context

Read `specs/CONVENTIONS.md` (ID rules, kinds, the `[NEEDS CLARIFICATION]` convention), `specs/ARCHITECTURE.md` and `specs/DESIGN_SYSTEM.md` if present, and any existing `features/<n>/` touching the same domain (to find models/view-models to `depends-on`).

### 3. Question round

Ask **one question at a time**. Use multiple-choice when the answer space is finite; open-ended only when genuinely exploratory. Cover, roughly in order: persona & intent · trigger & goal · scope boundaries (in/out for v1) · constraints (auth, retention, privacy, perf) · dependencies. When you don't know and the user hasn't said, **do not guess** — ask, or drop a `[NEEDS CLARIFICATION: <question>]` marker and move on.

### 4. Approach round

Propose 2-3 approaches with tradeoffs; lead with your recommendation and why (single combined view vs. separate views; inline vs. modal; local-first vs. always-server; one entity vs. entity+join). Get explicit approval on the approach before writing files.

### 5. Author the folder

Create `features/<NNNN>-<slug>/` (next number, kebab slug) following `specs/CONVENTIONS.md`:

- **`NARRATIVE.md`** — persona, situation, what we're building, why it matters, what it is **not**.
- **`stories/<id>.md`** — one story per file via the `writing-user-stories` skill. ID `story.<feature>.<capability>`, an `Independent test:` line, scenarios with stable `scenario.<feature>.<capability>.<short-name>` sub-IDs. **Prioritize stories P1 / P2 / P3** — P1 is the MVP slice that ships value alone; later priorities layer on. (This is spec-kit's rigor on Workbench's data model.)
- **`models/<id>.md`** (`domain.<entity>`), **`view-models/<id>.md`** (`vm.<feature>.<view>`: state, actions, transitions), **`user-flow/<id>.md`** (`flow.<feature>.<action>`, optional), **`errors/<id>.md`** (`error.<domain>.<kind>`, one per user-observable failure).

Optionally give `NARRATIVE.md` a **Success Criteria** section: measurable, technology-agnostic outcomes ("a user completes export in under 3 steps"), not implementation targets. Mark every unspecified detail with `[NEEDS CLARIFICATION]` rather than guessing.

### 6. Self-review

Scan for: leftover placeholders (`<id>`, TODO); contradictions (NARRATIVE vs. stories; VMs referencing models that exist); scope creep (scenarios that belong to a different feature); ambiguity; and the `[NEEDS CLARIFICATION]` count. Fix inline.

### 7. Validate, then the review gate

Run `specify scan` — it enforces the structural contract (frontmatter, ID↔filename, kind taxonomy, the scenario join). Fix anything it flags. Then tell the user what was authored and the outstanding-clarification count, and **wait for approval**:

```
Feature folder authored: features/<NNNN>-<slug>/
- NARRATIVE.md · stories/ (N stories, M scenarios) · models/ (N) · view-models/ (N) · errors/ (N)
Outstanding clarifications: K  ·  specify scan: clean
```

Iterate until approved. Only then commit.

### 8. Hand-off

- Markers remain → `/speckit.clarify <slug>`.
- Otherwise → `/speckit.analyze <slug>` for cross-artifact consistency, then `/speckit.plan <slug> <target>`.

### 9. Commit

After approval, commit the spec content under the scoped-commits convention (`specify gate scope` enforces it). Prefix `spec:`. One commit for a small folder (`spec: author features/<NNNN>-<slug>`), or split by artifact kind for a large one — each commit leaving the folder internally consistent. Never bundle implementation code here.

## Key principles

One question at a time · multiple-choice when possible · **YAGNI ruthlessly** (no scenarios/fields/errors the capability doesn't need) · **mark, don't guess** · when behavior is unclear across targets, consult the reference target (`reference_target` in `.speckit/specs.json`); when it's unset, no target is privileged — ask.

## Red flags — stop and re-scope

| Symptom | Means |
| --- | --- |
| More than ~6 stories per feature | Feature too large; decompose. |
| A story with more than ~6 scenarios | Story too large; split. |
| Many domain models with no shared aggregate root | Two features bundled; decompose. |
| Scenarios reference UI elements ("the green button") | Implementation creep; rewrite from intent. |
| `[NEEDS CLARIFICATION]` count > 10 after one round | Idea isn't clear yet; loop back to questions. |

## Related skills

- `writing-user-stories` — the per-story discipline this skill calls.
- `implementing-a-spec` — consumes the folder this produces, via `/speckit.implement`.
