# Developing SpecKit

SpecKit is a single Go module. The user-facing docs live in the [README](README.md)
and [docs/](docs/); this file is for working *on* the tool.

## Before you push

Run the local CI gate — it mirrors the `go` GitHub workflow exactly, plus a
gofmt check:

```sh
mise run ci   # gofmt-check + go build + go vet + go test, all packages
```

`mise run fmt` formats the tree in place. **Always run `mise run ci` before
pushing** so failures are caught locally instead of on a remote runner.

## Layout

- `cmd/specify/` — the CLI (Cobra). `main.go` wires subcommands; `render.go` does
  the Lip Gloss output.
- `internal/engine/` — the spec engine: `scan`, `verify`, `lock`, `drift`,
  `cover`, `parity`, `gate`.
- `internal/specmodel/` — the mechanized form of `specs/CONVENTIONS.md` (frontmatter,
  kinds, IDs, the scenario join).
- `internal/config/` — the `.speckit/specs.json` loader (targets).
- `internal/project/` — `init` scaffolding and the per-agent projection adapters,
  plus the skill / subagent / pack projection.
- `internal/coreassets/templates/` — the embedded assets `init`/`packs` project:
  `commands/`, `skills/`, `agents/`, `packs/<stack>/`.

## Conventions

- The fork dogfoods itself: its behavior is specified under `specs/` and
  `features/`, and its tests carry `// SPEC:` reverse pointers. `specify scan`
  must stay clean.
- Commits are scoped (`<scope>: <subject>`); `specify gate scope` enforces it.
- Projection changes are covered by golden trees under
  `internal/project/testdata/goldens/` — regenerate with
  `go test ./internal/project -run TestInitGoldenTrees -update`.
