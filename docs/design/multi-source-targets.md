# Design — multi-source verification targets

**Status:** approved / ready to implement. Let a target's `source` span more than
one directory so a Go-service target whose scenario tests live across several
packages (daemon, shared `internal/` packages, a sibling CLI) can be verified as
one unit.

## Thesis

Today `Target.Source` is a single string, so `specify verify <target>` scans
exactly one directory for scenario bindings. That's wrong for a target whose
bound tests are spread across packages. Trove's `go-service` target spans
`cmd/troved`, `internal`, and `cmd/trove-transcode`; narrowing to `cmd/troved`
misses bindings in `internal/organize`, `internal/medianame`, `internal/watch`,
and pointing at `.` is too broad (it would pull unrelated app/web bindings into a
Go-service verification).

The fix is a backward-compatible widening: `source` accepts **either** a string
(unchanged) **or** an array of directories. Verification scans every listed root
and joins the bindings.

```jsonc
// legacy — still valid, behaves exactly as today
"source": "apps/trove/app"

// new — multiple roots, joined into one target
"source": ["cmd/troved", "internal", "cmd/trove-transcode"]
```

## Where source is modeled, and the boundary

Two structs hold a `source`, in two layers that do **not** import each other:

- `config.Target.Source` — parsed from `.speckit/specs.json`, so it needs the
  flexible string-or-array decode.
- `engine.VerifyConfig.Source` — built **in-process** by the CLI
  (`cmd/specify`'s `verifyConfigFor`), never JSON-decoded. Confirmed by grep:
  the only constructions are literals in the CLI.

So the flexible JSON behavior belongs in `config` alone. The engine takes a plain
`[]string`. The CLI is the glue that maps `config.SourcePaths` → `[]string`. This
keeps the existing engine↔config independence (neither imports the other).

## Component 1 — `config.SourcePaths`

A small compatibility type rather than a breaking schema change:

```go
type SourcePaths []string
```

- **`UnmarshalJSON`** accepts a JSON string (`"x"` → `["x"]`) or a JSON array of
  strings (`["a","b"]` → as-is). Each entry is trimmed of surrounding
  whitespace as it's read. Any other JSON shape (number, object, array of
  non-strings) is a clear unmarshal error.
- **`MarshalJSON`** is *ergonomic*: a single path serializes back as a bare
  string, multiple paths as an array. Existing single-source configs round-trip
  byte-for-byte the same shape when SpecKit rewrites the file (`target add`,
  `deploy add`, `secrets sync`), so this change causes **zero churn** on configs
  in the wild.
- **`Validate(target string) []error`** — an empty list is "missing source dir"
  (same message class as today); any blank/whitespace-only entry is its own
  explicit error.
- **`First() string`** — the first path (or `""`). Used only by the deploy/secrets
  app-dir heuristic, which is single-app by nature.

`Target.Source` becomes `SourcePaths`. `Target.Validate` delegates its source
checks to `SourcePaths.Validate`.

## Component 2 — engine scans every root

- `VerifyConfig.Source` becomes `[]string`.
- Add two thin aggregators next to the existing single-dir scanners:
  - `ScanBindingsMany(root string, paths []string) ([]Binding, error)`
  - `ScanDeviationsMany(root string, paths []string) (map[SpecID]string, error)`

  Each loops the configured paths, calls the existing `ScanBindings(dir)` /
  `ScanDeviations(dir)` per `filepath.Join(root, path)`, and concatenates
  (deviations merge into one map). The single-dir functions stay as-is — they're
  the natural unit and keep their existing tests.
- `joinTarget` (verify) and `Parity` switch to the `*Many` variants.

Semantics that are unchanged because they operate on the *joined* binding set,
which doesn't care how many roots produced it:

- Binding identity, file, and line metadata — files may now come from any root.
- `bindings: "scoped"` still drops untagged tests; `strict` still flags them.
- A **dangling** binding (to an undeclared scenario) from *any* root still fails.
- Locking is unchanged: a spec locks only when all its in-scope scenarios passed
  and source integrity is clean.

`parity` is included even though the handoff doc only named verify: it reads
`cfg.Source` through the identical path, so `specify parity <multi-source-target>`
would otherwise break.

## Component 3 — CLI wiring (`cmd/specify`)

- `verifyConfigFor`: `Source: []string(t.Source)`.
- `target add` (scaffold path): scaffolds emit one source string → wrap as
  `config.SourcePaths{rt.Source}`.
- `targetRegisterCmd` / `registerTarget`:
  - `--source` becomes **repeatable** (`StringArrayVar`). Zero `--source` flags →
    fall back to the manifest's single derived source; one or more → use them
    verbatim. This lets a user author Trove's multi-source target without
    hand-editing JSON, though hand-editing remains fully supported.
  - The completeness check becomes "at least one source path".
- `deploy.go` / `secrets.go`: `filepath.Dir(t.Source.First())`.

The scaffold layer (`internal/scaffold`) is untouched: a generated target is
always single-source, and `RenderedTarget.Source` stays a string. No projected
assets change, so no golden-manifest regeneration is needed.

## Component 4 — docs

`docs/config.md` and the README target examples gain the array form alongside the
string form.

## Tests (TDD)

Config (`internal/config`):

1. Loads a legacy string `source` (→ one path, behaves as today).
2. Loads an array `source` (→ N paths).
3. Validation rejects missing source, empty array, and blank/whitespace entries.
4. `MarshalJSON` round-trip: one path → string, multiple → array.

Engine (`internal/engine`):

5. Verify joins bindings from two separate source roots into one target. Fixture:
   two spec scenarios in the spec library; one bound Go test in `cmd/example`;
   one bound Go test in `internal/example`; a gotest JSON report containing both
   passing identities → both scenarios pass and the target locks the spec.
6. A dangling binding from *either* root still fails verification.
7. `bindings: "scoped"` still ignores untagged tests while preserving the
   multi-source joined bindings.

CLI (`cmd/specify`):

8. `target register` with repeated `--source` writes an array-form target.

## Touch points (full enumeration)

- `internal/config/config.go` — `SourcePaths` type + `Target.Source` + validation.
- `internal/engine/verify_run.go` — `VerifyConfig.Source []string`; `joinTarget`
  uses `ScanBindingsMany`.
- `internal/engine/verify.go` — add `ScanBindingsMany`.
- `internal/engine/parity.go` — add `ScanDeviationsMany`; `Parity` uses it.
- `cmd/specify/main.go` — `verifyConfigFor`, `target add` wrap, `register` flag +
  completeness check.
- `cmd/specify/deploy.go`, `cmd/specify/secrets.go` — `t.Source.First()`.
- `docs/config.md`, `README` — array-form examples.

## Out of scope / non-goals

- Modeling products/contracts as first-class config (tracked separately).
- Per-source binding modes (one `bindings` mode still governs the whole target).
- Changing scaffold output to emit multi-source targets (scaffolds are
  single-source by construction).
