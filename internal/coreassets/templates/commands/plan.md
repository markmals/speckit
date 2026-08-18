---
description: Produce a per-target technical plan for a feature — how this target realizes the spec, with research, data shapes, and contracts.
---

# /speckit.plan — Technical plan for a target

The feature folder says **what** must hold, target-agnostically. A plan says **how one target realizes it** — the technology choices, the per-target design decisions, the test setup. One feature, N plans (one per target); the spec stays the single source of truth.

**Arguments:** `<feature> <target>` — e.g. `/speckit.plan 0001-managing-items app`.

## Prerequisites

The feature must be READY: `specify scan` clean and no open `[NEEDS CLARIFICATION]` markers (run `/speckit.analyze <feature>` if unsure). Load the **constitution** (`.speckit/memory/constitution.md`) if present — it is a hard gate. Any plan decision that violates a constitutional principle is an ERROR: stop and surface it, don't plan around it.

## Workflow

Write `features/<NNNN>-<slug>/plans/<target>.md`, working in two phases:

1. **Resolve unknowns (research).** For each open technical question — library choice, integration shape, target constraint — decide and record it as **Decision / Rationale / Alternatives considered**. Don't carry unknowns into the design.
2. **Design.** Map the feature's `domain.*` models to this target's data shapes; define the interface contracts the target exposes or consumes (drop this for a purely internal tool); note the test framework and the **binding affordance** this target uses for scenario IDs per `specs/CONVENTIONS.md` (Swift traits / MSTest `[TestProperty]` / kotlin `@Tag` / a `// [scenario.id]` comment). Optionally add a short runnable validation guide.

Keep the plan to decisions and shapes — it is not implementation, and it does not restate the spec.

## Hand-off

→ `/speckit.tasks <feature> <target>` to derive the ordered work, or straight to `/speckit.implement <feature> <target>` for a small feature. Commit the plan prefix `plan:` (or `<target>:`), scoped per `specify gate scope`.
