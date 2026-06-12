---
description: Generate a domain-focused requirements-quality checklist for a feature — "unit tests for the spec", not for the code.
---

# /speckit.checklist — Quality checklist

Generate a checklist that tests the **quality of the requirements**, not the behavior of the code. Think "unit tests for English": each item interrogates whether the spec is complete, clear, consistent, and measurable for a given concern. The output gates `/speckit.implement`.

**Arguments:** `<feature> <domain>` — domain is the focus area, e.g. `security`, `ux`, `api`, `performance`.

## Workflow

1. Read the feature folder (and `plans/<target>.md` if relevant to the domain) and the constitution if present. Ask up to 3 clarifying questions to calibrate **scope**, **depth**, and **audience** — only if the feature's signals leave them genuinely open.
2. Generate items across the quality taxonomy as it applies to the domain: **completeness · clarity · consistency · measurability · coverage · edge cases · non-functional requirements · dependencies · ambiguities & conflicts**.
3. Write `features/<NNNN>-<slug>/checklists/<domain>.md`. If the file exists, **append**, continuing the `CHK###` numbering — don't replace.

## What a good item looks like

Each item asks whether the **requirement** is sound — never whether the implementation works:

- **Do** — `CHK012 — Are the failure modes for export specified for an empty dataset? [story.export.csv]`
- **Don't** — `Verify the export button works` (that's a test of the code, not the spec).

Rules: phrase items as questions (`Are X defined/specified for Y?`); **≥80% must reference** a spec section ID or carry a `[Gap]` / `[Ambiguity]` / `[Conflict]` / `[Assumption]` tag; cap at ~40 items, merging near-duplicates. Never write items containing `Verify/Test/Confirm/Check + <behavior>` — that's testing code, not requirements.

## Hand-off

Report the path, item count, and focus. Resolve `[Gap]`/`[Ambiguity]` items via `/speckit.clarify` before `/speckit.implement` (which blocks on incomplete checklists). Commit prefix `spec:`, scoped per `specify gate scope`.
