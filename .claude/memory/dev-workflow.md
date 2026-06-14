---
description: SpecKit dev loop — the CI gate, regenerating golden init manifests, the dual exit-code convention, and scaffold .github gotchas.
---

# Dev workflow

**Gate before pushing.** Run `mise run ci` (build → vet → test, with a `gofmt`
check) before any push. It's the same trio CI runs.

**Golden init manifests.** `internal/project/golden_test.go` pins the exact file
tree `specify init` projects per agent (`testdata/goldens/init/<agent>.files.txt`).
Any change to projected assets — `templates/{skills,rules,agents,commands,memory}/`
or an adapter's projected paths — drifts these. Regenerate with:

```
go test ./internal/project -run TestInitGoldenTrees -update
```

Then eyeball the diff: it should contain *only* the files you intended to add/move.

**Dual exit-code convention** (CLI, `cmd/specify`). Two distinct failure modes:

- A real error (bad flag, missing file) is `return`ed from `RunE` → propagates to
  `main()`'s exit 1 (`SilenceUsage` keeps usage off).
- A "red but valid" finding (failing gate/parity, non-green verify) calls
  `os.Exit(1)` **after** writing its output. New commands must pick the right one
  deliberately. Outward, hard-to-undo actions (issue create, board moves, ruleset
  provisioning) should confirm resolved owner/repo first — mirrors the reconcile /
  `taskstoissues.md` guard.

**Scaffold `.github/` gotchas** (`internal/scaffold`). The per-stack `github/`
subtree must **double-nest**: `scaffolds/<stack>/github/.github/CODEOWNERS` →
repo-root `.github/CODEOWNERS` (the `github/` segment is stripped). `RenderGitHub`
is the *only* renderer with skip-existing=true, so re-running `target add` will
**not** update a stale PR template / CODEOWNERS — there's no overwrite path through
that seam. New `.tmpl` files are rendered *before* the skip check, so they must
parse against the fixed `scaffold.Data{Name,Dir,Product,Vars,Features}`. New embedded
files must live under `internal/coreassets/templates/` (the `//go:embed all:templates`
prefix is what pulls in dotdirs like `.github`).

**Verify scaffold combos SERIALLY, not in parallel.** The web scaffold's `lint`
runs `oxlint --type-aware` via `tsgolint` (a native typescript-go binary). Under the
CPU/memory contention of several concurrent `pnpm install` + `vite build` combos it
**flakily crashes** (`SIGSEGV`) or emits **false `TS2307: Cannot find module …`**
(while `tsgo` typecheck resolves the same module fine). Run combos one at a time —
or re-run `lint` once the parallel jobs finish (it goes 8/8 green serially). Don't
chase those failures as real lint errors. Heavy installs can run in the background;
just don't overlap the quality/lint steps.

See [[engine-boundaries]] for what the engine may read.
