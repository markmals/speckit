# 0001 — Spec engine

The deterministic core of `specify`: scan/verify/drift over the spec library (cover/parity arrive once a second target exists). These specs gate **Phase 3** of the fork plan, and the Phase 1 spike pressure-tests the `verify` scenario-join (D12) and `parity` deviation handling (D11) before this lands for real.

Scenarios are lifted from the plan's Phase-3 exit criteria.

| Spec | Capability |
| --- | --- |
| [`story.engine.scan`](stories/engine.scan.md) | Lint the spec library against the `domain.specmodel` invariants. |
| [`story.engine.verify`](stories/engine.verify.md) | Run a target's tests, normalize reports, join to scenarios, write the lock on green. |
| [`story.engine.drift`](stories/engine.drift.md) | Report specs whose content hash no longer matches their locked-green hash. |
| [`story.engine.parity`](stories/engine.parity.md) | Per (target × scenario) parity: conforming / declared-deviation / drifted / missing / suspect. |
| [`story.engine.cover`](stories/engine.cover.md) | Per-spec coverage across the target matrix, read from the lock. |
| [`story.engine.lock`](stories/engine.lock.md) | The single writer of the acknowledgment lock (D7). |
| [`story.engine.gate`](stories/engine.gate.md) | Agent-agnostic enforcement subchecks for git/CI (D8). |

Depends on [`domain.specmodel`](../../specs/models/specmodel.md) and [`conventions`](../../specs/CONVENTIONS.md).
