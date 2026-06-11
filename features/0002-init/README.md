# 0002 — Init & projection

Project bootstrapping and the per-agent command projection (D2/D4), plus the extension-install sugar that replaces Workbench's superset-then-prune `/setup` (D3). These specs gate **Phase 2** (`init`, projection) and **Phase 4** (`--platforms` sugar, extension round-trip).

| Spec | Capability |
| --- | --- |
| [`story.init.basic`](stories/init.basic.md) | `specify init` yields a working project projected for the chosen agent. |
| [`story.extension.install`](stories/extension.install.md) | `init --platforms` installs bundled packs; `extension add/remove` round-trips. |

Depends on [`conventions`](../../specs/CONVENTIONS.md).
