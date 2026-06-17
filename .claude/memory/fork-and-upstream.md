---
description: markmals/spec-kit is a FORK of github/spec-kit — gh and PR operations default to the upstream parent, so always target the fork explicitly.
---

# Fork & upstream

`markmals/spec-kit` (this repo) is a **fork** of `github/spec-kit` (GitHub's
official Spec-Kit). The fork relationship keeps biting agents who forget it:

- **`gh pr create` defaults the base to the upstream parent** (`github/spec-kit`),
  not this fork. It fails with `No commits between main and <branch>` /
  `Head sha can't be blank`, or silently tries to open a PR against GitHub's repo.
  Always target the fork explicitly:

  ```sh
  gh pr create -R markmals/spec-kit --base main --head <branch>
  ```

  Or set it once per clone so every `gh` command defaults correctly:

  ```sh
  gh repo set-default markmals/spec-kit
  ```

- Same `-R markmals/spec-kit` caveat for **any** cross-fork `gh` op (`gh pr list`,
  `gh issue`, `gh api`) — without it you read/write the upstream by accident.

- The git remote is `markmals/spec-kit`, but GitHub redirects the older name
  `markmals/speckit`, so `gh`/PR URLs may show either spelling — same repo.

PRs here are **fork-internal** (`markmals/spec-kit` `main` <- branch); we do not
PR up to `github/spec-kit`.
