# 0005 — Adoption

Bringing SpecKit into a repo whose code already exists: registering a target without touching it, choosing which target is the reference by configuration, and loading configs written by earlier versions.

| Spec | Capability |
| --- | --- |
| [`story.adoption.target-add`](stories/adoption.target-add.md) | `specify target add` records an existing target in config only — no files rendered, no scripts run. |
| [`story.adoption.reference-target`](stories/adoption.reference-target.md) | The reference target is read from `reference_target`; unset with several targets privileges none. |
| [`story.adoption.legacy-config`](stories/adoption.legacy-config.md) | Retired keys and older schema versions load with a notice, never a hard failure. |

Depends on [`story.engine.verify`](../0001-engine/stories/engine.verify.md) and [`conventions`](../../specs/CONVENTIONS.md).
