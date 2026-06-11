---
id: domain.catalog
kind: domain
depends-on: [domain.manifest]
---

# Domain: Catalog

The index that resolves an extension/preset id to something installable. Two catalogs: the **first-party** catalog (vendored in the release, resolvable offline) and a **snapshotted community** catalog (reference-only — community extensions are not promised to install, D6).

## Shape

A catalog is a set of entries, each:

- `id`, `name`, `version`, `kind`.
- `bundled` — true if the asset ships inside the binary/release (no download).
- `download_url` — present for non-bundled entries.
- `source_catalog` — `first-party` | `community-snapshot`.

Catalog fetches are cached on disk with an expiry.

## Invariants

- **C1 — Bundled needs no url.** A `bundled` entry resolves with no network; a non-bundled entry must carry a `download_url`.
- **C2 — No url, not bundled → error.** An entry that is neither bundled nor has a `download_url` is invalid and `scan`/resolution rejects it (mined from `tests/test_extensions.py::catalog_entries_without_urls_raises_error`).
- **C3 — First-party resolves offline.** The first-party catalog and every entry it marks bundled resolve with no network access.
- **C4 — Community is reference-only.** `community-snapshot` entries are listed for discovery but install attempts report reference-only (D6), never silently fail.
- **C5 — Cache expires.** A cached catalog past its expiry is refreshed (or, offline, used with a staleness note) — never served as fresh forever.
