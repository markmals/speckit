---
id: domain.work-item
kind: domain
depends-on: [conventions]
---

# Domain: Work item

One unit of in-flight coordination, as every provider presents it. A work item is **ephemeral coordination** — never an input to `scan`/`verify`/`drift`/`cover`/`parity`/`gate`, and never a source of spec truth; the durable record is the spec library and the lock.

## Shape

- `id` — provider-allocated, stable for the item's life.
- `title` — human description.
- `state` — one of the canonical states below.
- `type` — `task` | `defect`; empty means `task`.
- `spec` — optional; the spec id this item advances.
- `url` — provider-native link, when the provider has one.

A create request carries `title`, optional `type`, and optional `spec`.

## States

`ready` → `in-progress` → `done`, with `blocked` reachable from any non-done state. Every provider maps its native states onto exactly these four.

## Acceptance Criteria

- [scenario.work-item.canonical-states] The canonical states are exactly `ready`, `in-progress`, `blocked`, `done` — a provider exposes no fifth state and drops none of the four.
- [scenario.work-item.defect-is-a-type] A defect and a task differ only by `type`; both carry the same shape and move through the same states.
- [scenario.work-item.spec-pointer] The optional `spec` field names a spec id, linking the item to the spec it advances.
- [scenario.work-item.never-engine-input] The engine never reads work state: no engine command's output changes when work items are created, moved, or deleted.
