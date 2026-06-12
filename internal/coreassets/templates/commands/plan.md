---
description: Produce a per-platform technical plan for a feature — how this platform realizes the spec, with research, data shapes, and contracts.
---

# /speckit.plan — Technical plan for a platform

The feature folder says **what** must hold, platform-agnostically. A plan says **how one platform realizes it** — the stack, the per-platform design decisions, the test setup. One feature, N plans (one per platform); the spec stays the single source of truth.

**Arguments:** `<feature> <platform>` — e.g. `/speckit.plan 0001-managing-items web`.

## Prerequisites

The feature must be READY: `specify scan` clean and no open `[NEEDS CLARIFICATION]` markers (run `/speckit.analyze <feature>` if unsure). Load the **constitution** (`.speckit/memory/constitution.md`) if present — it is a hard gate. Any plan decision that violates a constitutional principle is an ERROR: stop and surface it, don't plan around it.

## Workflow

Write `features/<NNNN>-<slug>/plans/<platform>.md`, working in two phases:

1. **Resolve unknowns (research).** For each open technical question — library choice, integration shape, platform constraint — decide and record it as **Decision / Rationale / Alternatives considered**. Don't carry unknowns into the design.
2. **Design.** Map the feature's `domain.*` models to this platform's data shapes; define the interface contracts the platform exposes or consumes (drop this for a purely internal tool); note the test framework and the **binding affordance** this platform uses for scenario IDs per `specs/CONVENTIONS.md` (Swift traits / MSTest `[TestProperty]` / kotlin `@Tag` / a `// [scenario.id]` comment). Optionally add a short runnable validation guide.

Keep the plan to decisions and shapes — it is not implementation, and it does not restate the spec.

## Hand-off

→ `/speckit.tasks <feature> <platform>` to derive the ordered work, or straight to `/speckit.implement <feature> <platform>` for a small feature. Commit the plan prefix `plan:` (or `<platform>:`), scoped per `specify gate scope`.
