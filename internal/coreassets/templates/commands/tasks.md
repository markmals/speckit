---
description: Derive an ordered, story-prioritized task list for implementing a feature on a platform; each task maps to a spec ID.
---

# /speckit.tasks — Ordered work for a platform

Break a feature down into an executable, dependency-ordered task list for one platform. Each task names exactly **one spec ID** (a story or scenario) to realize, so the work maps straight onto reverse pointers and the `specify verify` join. This step is **optional** — for a small feature, `/speckit.implement` can take the spec IDs directly — but it earns its keep on multi-story or multi-platform work.

**Arguments:** `<feature> <platform>` — e.g. `/speckit.tasks 0001-managing-items ios`.

## Workflow

Read the feature folder and `plans/<platform>.md` (if present). Write `features/<NNNN>-<slug>/tasks/<platform>.md` as a checklist, phased:

- **Phase 1 — Setup:** project/test scaffolding this platform needs.
- **Phase 2 — Foundational:** shared models/services every story depends on.
- **Phase 3+ — one phase per user story, in P1 → P2 → P3 order.** Each phase realizes one story and is **independently testable** — P1 alone is a shippable MVP. List the story's scenarios as tasks.
- **Final phase — Polish:** cross-cutting cleanup.

Each task line: `- [ ] T### [P?] [US#] <spec-id> — <what to do> (<exact file path>)`.

- `[P]` marks tasks that touch **different files with no dependency** and may run in parallel; omit it when they'd collide.
- `[US#]` labels the owning story.
- Give an exact file path so the task is executable without re-deriving context.

End with the dependency notes, the parallelizable groups, and the **MVP scope** (usually US1 only). Don't write tests-as-tasks unless the work is genuinely test-first scaffolding — the RED test for each scenario is written inside `/speckit.implement`, by the `test-driven-development` discipline.

## Hand-off

→ `/speckit.implement <feature> <platform>` (follows these phases), or `/speckit.taskstoissues <feature> <platform>` to file them as GitHub issues. Commit prefix `plan:`/`<platform>:`, scoped per `specify gate scope`.
