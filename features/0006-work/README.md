# 0006 — Work tracking

Pluggable, ephemeral work coordination behind five verbs (`ready`, `create`, `claim`, `move`, `list`), with a committed-markdown default and a structural firewall keeping every provider out of the engine.

| Spec | Capability |
| --- | --- |
| [`domain.work-item`](models/work-item.md) | The work item: shape, four canonical states, two types, never an engine input. |
| [`story.work.roundtrip`](stories/work.roundtrip.md) | The same five verbs drive every provider; create lands in `ready`, claim moves to `in-progress`. |
| [`story.work.markdown`](stories/work.markdown.md) | The default provider: one committed markdown file, diffable, offline, no external binary. |
| [`story.work.providers`](stories/work.providers.md) | Provider selection, the `none` provider, beads and github-projects adapters, and the import firewall. |

Depends on [`conventions`](../../specs/CONVENTIONS.md).
