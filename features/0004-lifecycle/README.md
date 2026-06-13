# 0004 — Spec lifecycle

The regeneration loop and its trail: applying a spec to a target, reconciling when a target races ahead, and the ledger that records each attempt (D9). Plan/tasks are disposable per-(spec × target) execution artifacts; the durable state is the spec library + the lock + the ledger.

| Spec | Capability |
| --- | --- |
| [`story.lifecycle.apply`](stories/lifecycle.apply.md) | `apply <spec> <target>` — failing tests first, then impl, then verify; disposable plan/tasks. |
| [`story.lifecycle.reconcile`](stories/lifecycle.reconcile.md) | `reconcile <target>` — fold a target's lead back into the spec + others (human-approved). |
| [`story.lifecycle.ledger`](stories/lifecycle.ledger.md) | `apply` appends a run record; the ledger is the bench's raw material. |

Depends on the engine ([`story.engine.verify`](../0001-engine/stories/engine.verify.md), [`story.engine.drift`](../0001-engine/stories/engine.drift.md)) and [`domain.ledger`](../../specs/models/ledger.md).
