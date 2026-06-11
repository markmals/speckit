# 0002 — Init & projection

Project bootstrapping and the per-agent command projection (D2/D4), plus the extension-install sugar that replaces Workbench's superset-then-prune `/setup` (D3). These specs gate **Phase 2** (`init`, projection) and **Phase 4** (`--platforms` sugar, extension round-trip).

| Spec | Capability |
| --- | --- |
| [`story.init.basic`](stories/init.basic.md) | `specify init` yields a working project projected for the chosen agent. |
| [`story.init.projection`](stories/init.projection.md) | The command set projects per agent (claude/codex/copilot/generic); `.speckit/` runtime, no scripts/workflows. |
| [`story.extension.install`](stories/extension.install.md) | `init --platforms` installs bundled packs; offline catalog; community reference-only. |
| [`story.extension.add`](stories/extension.add.md) | `extension add` from catalog / URL / `--dev`, with priority. |
| [`story.extension.remove`](stories/extension.remove.md) | `extension remove` restores overridden state; add→remove round-trips. |
| [`story.preset.apply`](stories/preset.apply.md) | `--preset` / `preset apply` installs a curated bundle. |

Golden manifests for the projection are captured from the oracle in [`testdata/oracle-init/`](../../testdata/oracle-init/) (D14). Depends on [`conventions`](../../specs/CONVENTIONS.md).
