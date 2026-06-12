---
description: Read-only cross-artifact consistency and quality analysis of a feature; reports findings by severity, never edits.
---

# /speckit.analyze — Consistency check

A strictly **read-only** pass over a feature folder: confirm the artifacts are internally consistent, every reference resolves, and the feature is ready to implement. Output a report; never edit (route fixes to `/speckit.clarify`, a manual edit, or `/speckit.implement`).

**Arguments:** `<feature>` — id or slug.

## Two layers

1. **Mechanized (the engine).** Run `specify scan` first. It owns the structural contract: frontmatter present and well-formed, ID matches filename stem, `kind` in the taxonomy, files in the right directory, and the scenario↔test join is total (no unbound scenario, no test bound to a missing scenario). Treat every `specify scan` error as **CRITICAL** — don't re-litigate it, just report it.

2. **Semantic (you).** On top of the scan, run these detection passes:
   - **Coverage** — NARRATIVE substantive; `stories/` non-empty; a `models/` entry for every entity referenced; a `view-models/` entry for every view a story implies; an `errors/` entry for every failure mentioned.
   - **Reference integrity** — every `depends-on` and inline ID resolves in `features/` or `specs/`.
   - **Story/scenario consistency** — each story has ≥1 scenario; each scenario has a unique `<!-- id: scenario.* -->`; each story has an `Independent test:` line and a P1/P2/P3 priority.
   - **VM/domain alignment** — each view-model `depends-on` its domain models; its actions map to story user-actions; its state maps to domain fields or derivations.
   - **Duplication / ambiguity / underspecification / inconsistency** — conflicting requirements, vague terms, gaps two readings could fill differently.
   - **Outstanding clarifications** — count `[NEEDS CLARIFICATION]` markers; any remaining ⇒ NOT READY.

## Report

Findings table (severity **CRITICAL / HIGH / MEDIUM / LOW**, location, what's wrong, suggested route), a coverage summary, the open-clarification count, and a final **READY / NOT READY for /speckit.implement** line with the single suggested next action. Cap at ~50 findings; if more, say so. Make no edits.

## Hand-off

Markers or gaps → `/speckit.clarify`. Clean → `/speckit.plan <feature> <target>`.
