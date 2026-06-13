---
description: The offline determinism line — what the engine may and may not read, and where GitHub/network code lives.
---

# Engine boundaries

**The offline determinism line is the core invariant.** The engine commands —
`scan` / `verify` / `lock` / `drift` / `cover` / `parity` / `gate` (in
`internal/engine` + `internal/specmodel`) — must never read GitHub or the network.
Correctness is repo-local and offline. This holds *structurally*, not by convention:

- `internal/engine` and `internal/specmodel` import **no** `net/http`, GitHub SDK,
  or GraphQL code. Keep it that way.
- `specmodel.LoadLibrary` walks **only** `specs/` and `features/`. Agent dirs
  (`.claude/`, `.agents/`, `.github/`) and `.speckit/` runtime can never enter the
  spec model. Agent memory is invisible to the engine by construction.
- The source walker (`internal/engine/walk.go`) is git-free/file-based; it skips
  `.git`, `node_modules`, and bare dir names from `.gitignore`. Don't add a git
  dependency to stay offline.

**All GitHub/network code lives in `internal/github`**, imported **only** by
`cmd/specify` command constructors — never by `internal/engine` or
`internal/specmodel`. That one rule is what preserves the offline guarantee as the
GitHub-native surface grows. Board-sync / issue calls must never become an input to
verify/lock/drift/cover/parity/gate, and a board-sync failure must never block a
local `verify`.

The lock (`.speckit/lock/`) is the only durable engine truth; only `Lock`/`Verify`
write it, and `gate generated` guards it. GitHub Issues/Projects are *ephemeral
coordination*, not truth — see [[dev-workflow]] for the gate, and
`docs/design/github-integration.md` for the determinism-line table.
