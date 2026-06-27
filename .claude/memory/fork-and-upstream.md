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

- **Canonical name is `markmals/speckit`; `markmals/spec-kit` is an OLDER name that
  GitHub rename-redirects to it** — ONE repo (not two), so both spellings resolve in
  `gh` and `git`. Proof: `git push` to the `spec-kit` URL prints "repository moved →
  speckit.git"; `gh api repos/markmals/spec-kit` returns `full_name: markmals/speckit`;
  a PR opened with `-R markmals/spec-kit` lands at `markmals/speckit#N`. (Earlier this
  note had the direction backwards — `speckit` is the new/current name, not the old one.)
  Prefer `markmals/speckit` in new commands; `spec-kit` keeps working via the redirect.

- **Releases all target the canonical `speckit`.** goreleaser's `release.github.name:
  speckit`, the `release.yml` Homebrew-tap dispatch, and the Mise plugin's download URL
  all point at `markmals/speckit` — consistent with the canonical name, so a `v*` tag
  pushed from this clone publishes to the right place with the repo-scoped `GITHUB_TOKEN`.

PRs here are **fork-internal** (`markmals/speckit` `main` <- branch); we do not
PR up to `github/spec-kit`.
