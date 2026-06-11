---
id: domain.lock
kind: domain
depends-on: [domain.specmodel]
---

# Domain: Acknowledgment lock (D7)

The drift-state record. Replaces Workbench's mtime invariant wholesale (git doesn't preserve mtimes; CI clones and worktrees make it vacuous). Sharded per spec so parallel worktree agents never merge-conflict in it.

## Shape

One shard file per `(platform, spec-id)` at `.speckit/lock/<platform>/<spec-id>`:

- `spec_hash` — the content hash of the spec version last verified green on this platform.
- `scenarios` — per-scenario result captured at that hash: `{ scenario-id: pass | fail }`.
- `verified_at` — timestamp of the green run (informational; not used for drift).

## Invariants

- **L1 — Single writer.** Only `specify lock` writes a shard, and only `specify verify` invokes it, only on green. Nothing else mutates the lock.
- **L2 — Content hash, not mtime.** `spec_hash` is computed from spec file content; drift is hash-mismatch-or-missing, never an mtime comparison (D7).
- **L3 — Sharded.** One file per spec per platform — never one combined lockfile — so concurrent worktrees writing different specs don't conflict.
- **L4 — Generated.** The lock path is covered by the generated-file gate (`story.engine.gate`); hand-edits are blocked.
