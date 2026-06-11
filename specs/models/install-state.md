---
id: domain.install-state
kind: domain
depends-on: [domain.manifest]
---

# Domain: Install state

The record, under `.speckit/`, of what is installed in a project and how to undo it — the data that makes `extension add`/`remove` a clean round-trip (D3).

## Shape

A list of installed entries, each:

- `id`, `version`, `kind` — from the manifest.
- `source` — `bundled` | `url:<u>` | `dev:<path>`.
- `priority` — integer; higher wins when two extensions provide the same command.
- `files` — the paths this extension wrote (so removal knows what to delete).
- `overrides` — for each command this install shadowed, the prior owner + its content, so removal can restore it.

## Invariants

- **S1 — One entry per install.** Every installed extension has exactly one state entry; removing it deletes the entry.
- **S2 — Restorable overrides.** If installing shadowed an existing command, the prior version is captured in `overrides` so `extension remove` restores it (see `story.extension.remove`).
- **S3 — Files owned.** `files` lists only paths this extension wrote; removal never deletes a path another entry owns.
- **S4 — Round-trip identity.** Applying then reverting an install returns both the file tree and the install state to their prior values.
