---
description: Create or update the project constitution — the non-negotiable principles — and propagate changes to templates, commands, and conventions.
---

# /speckit.constitution — Project principles

The constitution at `.speckit/memory/constitution.md` is the single authority for a project's non-negotiable principles — the rules `/speckit.plan` and `/speckit.implement` treat as hard gates. This command creates or amends it and keeps everything that depends on it in sync.

**Arguments:** `[principle text]` — optional. Partial updates or full content; otherwise work interactively.

## Workflow

1. **Load** `.speckit/memory/constitution.md`; identify any `[ALL_CAPS_TOKEN]` placeholders. Collect values from the user's input or the repo's existing conventions.
2. **Draft** the update. Each principle has a name, its non-negotiable rule(s) in normative language (**MUST / SHOULD**), and a short rationale. Complete the governance section (amendment procedure, versioning policy).
3. **Version** the change semantically: **MAJOR** = a principle removed or redefined incompatibly; **MINOR** = a new principle or materially expanded guidance; **PATCH** = wording/clarification. Keep `RATIFICATION_DATE` (first adopted) distinct from `LAST_AMENDED_DATE`.
4. **Propagate.** Re-read what depends on the constitution and flag anything now stale: the templates under `.speckit/templates/`, the `/speckit.*` command prompts, `specs/CONVENTIONS.md`, and the process skills. List the follow-ups; don't silently leave drift.
5. **Record.** Prepend a Sync Impact Report (HTML comment) at the top of the file: version change, sections added/modified/removed, and propagation status.
6. **Validate.** No unexplained `[TOKEN]` remains (use `TODO(<FIELD>): why` for genuinely-unknown governance fields); dates ISO 8601; language is normative.

## Hand-off

Report the version bump and its rationale, the files needing manual follow-up, and a suggested commit message. Commit prefix `spec:` (or `chore:` for wording-only), scoped per `specify gate scope`.
