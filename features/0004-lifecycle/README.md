# 0004 — Spec lifecycle

The regeneration loop and its trail: applying a spec to a platform, reconciling when a platform races ahead, and the ledger that records each attempt (D9). Plan/tasks are disposable per-(spec × platform) execution artifacts; the durable state is the spec library + the lock + the ledger.

| Spec | Capability |
| --- | --- |
| [`story.lifecycle.apply`](stories/lifecycle.apply.md) | `apply <spec> <platform>` — failing tests first, then impl, then verify; disposable plan/tasks. |
| [`story.lifecycle.reconcile`](stories/lifecycle.reconcile.md) | `reconcile <platform>` — fold a platform's lead back into the spec + others (human-approved). |
| [`story.lifecycle.ledger`](stories/lifecycle.ledger.md) | `apply` appends a run record; the ledger is the bench's raw material. |

Depends on the engine ([`story.engine.verify`](../0001-engine/stories/engine.verify.md), [`story.engine.drift`](../0001-engine/stories/engine.drift.md)) and [`domain.ledger`](../../specs/models/ledger.md).
