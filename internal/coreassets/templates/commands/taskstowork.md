---
description: File a feature's target task list as tracked work items through the configured work provider.
---

# /speckit.taskstowork — Tasks → work items

Turn `features/<NNNN>-<slug>/tasks/<target>.md` into tracked work items so multi-session work is claimable and observable. Everything goes through `specify work`, which talks to whatever provider `.speckit/specs.json` configures under `work.provider` (`markdown` | `beads` | `github-projects` | `none`; default `markdown`) — never talk to a tracker directly.

**Arguments:** `<feature> <target>` — the task list to file.

## Workflow

1. **Guard the provider.** Run `specify work list --json`. If it reports no work provider configured, stop and tell the user — they opted out with `work.provider: none`. Otherwise confirm the task count with the user before creating anything.
2. **Read** `features/<NNNN>-<slug>/tasks/<target>.md`. Each task line becomes one work item.
3. **Create**, one item per task: `specify work create "<spec-id> — <task summary> (<target>)" --spec <spec-id>`. Add `--type defect` when the task fixes broken behavior; everything else stays the default `task`.
4. **Report** the created item IDs (and URLs where the provider returns them). Note that `[P]` tasks are parallelizable and `[US#]` marks the owning story. New items land in `ready`; whoever picks one up runs `specify work claim <id>`, and state moves with `specify work move <id> <state>` through the canonical states `ready` → `in-progress` → `blocked` → `done`.

## Discipline

Outward action — creating work items is visible to the whole team, so confirm the task count before the first `specify work create`. File the list as-is; don't invent tasks not in the file. One task, one item. Work items are ephemeral coordination, never spec truth — the spec library stays authoritative, and the engine (`scan`/`verify`/`drift`/`cover`/`parity`/`gate`) never reads them.
