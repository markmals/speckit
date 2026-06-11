---
id: domain.specmodel
kind: domain
depends-on: [conventions]
---

# Domain: Spec Model

The data model of a spec itself — the thing `internal/specmodel` parses and every engine command (`scan`, `verify`, `drift`, `cover`, `parity`) operates over. This is `CONVENTIONS.md` mechanized.

## Shape

A **Spec** is a markdown file with YAML frontmatter:

- `id` — a `SpecID`: dotted, lowercase, hierarchical, stable. Required. Must equal the filename stem (dots preserved).
- `kind` — one of the closed `Kind` taxonomy. Required.
- `depends-on` — flat list of `SpecID` (optional; non-transitive).
- `status` — `draft | accepted`. Optional; defaults to `accepted`.

A **Scenario** is a Gherkin block inside a `story` spec, carrying a sub-ID of the form `scenario.<feature>.<capability>.<short-name>` declared in an HTML comment (`<!-- id: ... -->`).

A **reverse pointer** is a `// SPEC: <id>` comment in implementation or test code, optionally `(deviates: <reason>)` or the literal `// SPEC: manual`.

## Invariants

- **I1 — ID/filename agreement.** A spec's `id` trailing segment equals its filename stem.
- **I2 — Closed kind.** `kind` is a member of the taxonomy; an unknown kind is invalid.
- **I3 — Prefix agreement.** `id` begins with `kind`'s ID prefix (e.g. a `domain` spec's id starts `domain.`). Singular cross-cutting kinds (`architecture`, `design-system`, `conventions`) have no prefix and their id is the kind name.
- **I4 — ID uniqueness.** No two specs in the library share an `id`.
- **I5 — Pointer integrity.** Every `depends-on` entry resolves to a spec that exists; every `[scenario.<id>]` test tag resolves to a declared scenario sub-ID (D12).
- **I6 — Scenario sub-IDs.** Every Gherkin scenario in a `story` declares a sub-ID.

## Validation rules (consumed by `specify scan`)

`scan` lints the library against I1–I6 and reports, per finding: the offending file, the violated invariant, and a fix hint. Dangling `depends-on` (I5), duplicate IDs (I4), filename↔ID mismatch (I1), unknown kind (I2), prefix mismatch (I3), and scenarios missing sub-IDs (I6) are all hard findings — `scan` exits non-zero when any are present.

## Notes

Platform is **not** part of the model: a `SpecID` describes abstract behavior, and platform divergence is captured only in reverse pointers via `(deviates:)`. The lock and parity state (D7/D11) are keyed by `(platform, SpecID, content-hash)` but live outside the spec file, under `.speckit/`.
