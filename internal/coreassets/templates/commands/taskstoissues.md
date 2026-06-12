---
description: File a feature's platform task list as GitHub issues on the repo matching the git remote.
---

# /speckit.taskstoissues — Tasks → GitHub issues

Turn `features/<NNNN>-<slug>/tasks/<platform>.md` into GitHub issues so the work is trackable in the same place SpecKit's git workflow already lives (worktrees, PRs, Projects — see the README).

**Arguments:** `<feature> <platform>` — the task list to file.

## Workflow

1. **Guard the target.** Resolve the GitHub repo from the git remote (`gh repo view --json nameWithOwner`). If the remote is not a GitHub URL, **stop** — never create issues in the wrong repo. Confirm the resolved `owner/repo` with the user before creating anything.
2. **Read** `features/<NNNN>-<slug>/tasks/<platform>.md`. Each task line becomes one issue.
3. **Create**, one issue per task, with `gh issue create`:
   - Title: `<spec-id> — <task summary> (<platform>)`.
   - Body: the task detail, the file path, and a link to the spec file it realizes.
   - Labels: the platform and the owning story (`US#`) where the repo has them; create labels only if the user approves.
4. **Report** the created issue numbers + URLs. Note that `[P]` tasks are parallelizable and `[US#]` groups map to milestones if the user wants them.

## Discipline

Outward action — creating issues is visible and hard to undo, so confirm the repo and the task count before the first `gh issue create`. File the list as-is; don't invent tasks not in the file. One task, one issue.
