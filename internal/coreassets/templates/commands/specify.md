---
description: Author a new feature folder from a description — narrative, prioritized stories, models, view-models, and errors.
---

# /speckit.specify — Author a feature

Turn a natural-language feature description into a populated `features/<NNNN>-<slug>/` folder, following `specs/CONVENTIONS.md`. The feature folder **is** the spec; there is no separate `spec.md`/`plan.md`/`tasks.md` at this stage — the engine joins behavior to tests off the ID'd artifacts you author here.

**Arguments:** `<feature description>` — free text. (Everything after the command is the description.)

## What to do

Run the **`brainstorming-feature`** skill with the user's description. It carries the discipline: scope check → context → one-question-at-a-time → approaches → author → self-review. Honor its **hard gate** — no implementation until the folder has a substantive NARRATIVE, ≥1 story with scenarios, and the user's explicit approval.

Fold in this structural rigor as you author:

- **Prioritize stories P1 / P2 / P3.** P1 is the MVP slice that delivers value alone; later priorities layer on. Each story carries an `Independent test:` line.
- **Measurable, technology-agnostic success criteria** in the NARRATIVE — "a user completes export in under 3 steps", never "p95 < 200ms" or a framework name.
- **At most 3 `[NEEDS CLARIFICATION]` markers**, prioritized scope > security/privacy > UX > technical. Mark, don't guess.
- Each story's scenarios get stable `scenario.<feature>.<capability>.<short-name>` sub-IDs (via the `writing-user-stories` skill) — these are what tests bind to.

## Validate before handing back

Run `specify scan`. It enforces the structural contract (frontmatter, ID↔filename, kind taxonomy, the scenario join) — fix anything it flags. Then report what was authored, the `specify scan` status, and the open-clarification count, and wait for approval.

## Hand-off

- Clarifications remain → `/speckit.clarify <slug>`.
- Clean → `/speckit.analyze <slug>`, then `/speckit.plan <slug> <target>`.

Commit only after approval, prefix `spec:`, scoped per `specify gate scope`. Never bundle implementation code.
