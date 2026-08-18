---
id: domain.manifest
kind: domain
depends-on: [conventions]
---

# Domain: Extension manifest

The manifest that describes an extension or integration — the fork's own format, seeded from upstream's `extension.yml` schema (D6) with fork fields namespaced under a `speckit:` key.

## Shape

- `id` — canonical, unique within a catalog (kebab-case).
- `name` — human display name (used by `extension add "<Display Name>"`).
- `version` — semver.
- `kind` — `extension` | `integration` | `preset`.
- `requires.speckit_version` — **advisory** (D6): a mismatch warns, never blocks.
- `provides` — the assets it projects: `commands`, `skills`, `templates`, `scripts` (git-hook trampolines only).
- `speckit:` — fork-namespaced fields: `verify` (the extension's verify-adapter config).

## Invariants

- **M1 — Unique id.** No two manifests in a resolved catalog share an `id`.
- **M2 — Advisory version.** `requires.speckit_version` never blocks install; an out-of-range value produces a warning only.
- **M3 — Provides resolve.** Every entry under `provides` points at an asset that exists in the extension.
- **M4 — Namespaced fork fields.** Fork-specific data lives only under `speckit:`; the rest of the manifest stays schema-compatible with upstream so community manifests parse (they just don't install — reference-only, D6).
