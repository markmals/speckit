---
description: Mise monorepo invariant; the unstable-parser comment-preserving merge; family↔member drift coupling; tuist placement.
---

# Mise monorepo

## The invariant

A SpecKit repo is a **mise monorepo from the first `specify target add`**. The
orchestrator (`wireMonorepo` in `cmd/specify/monorepo.go`) runs after every
`target add` / `target register` and maintains the root `mise.toml`:

- **Member #1** → creates the root config with `monorepo_root = true`,
  `[monorepo].config_roots` (single-level globs like `apps/*`; filesystem
  auto-discovery is deprecated and warns), and the family's `[tools]` pins.
  Member keeps all task bodies **inline** (self-contained, clean for a
  single-member repo).
- **Member #2 (same family)** → "promotion": hoists the family's canonical
  task bodies to root `[task_templates]` and converts each member's still-canonical
  inline `run` to `extends = "<family>:<task>"`. Only tasks whose `run` still
  matches the canonical (after the member's own `[vars]` substitution) are
  converted; user-edited tasks are left inline.

`target.command` in `.speckit/specs.json` uses the native monorepo form:
`mise //{{.Dir}}:test` (not `cd {{.Dir}} && mise run test`). The `mise //dir:task`
invocation runs with cwd = the member dir, so `report: {{.Dir}}/junit.xml` still
resolves correctly from the project root.

Families today: `node` (`web`), `swift` (`apple`, `swift-package`, `swift-cli`),
`go` (`go-service`). A `node` second member is reachable via a second `web` target;
a `swift` second member by e.g. `apple` + `swift-package`.

## The merge engine (`go-toml/v2` `unstable`)

Root config edits use `github.com/pelletier/go-toml/v2` v2.3.1's `unstable`
byte-range parser — **pinned at v2.3.1 because the `unstable` API can shift across
minor bumps**. This preserves every comment and user edit (no managed-region
markers).

Critical parser gotcha: **`Table` nodes have a zero `Raw` field** — do not use
`e.Raw` to locate a table header. Instead derive the header span from the `Key`
child: find the first key's `Raw.Offset`, scan back for `[`, then forward for
`\n`. `KeyValue` nodes have a correct non-zero `Raw` covering the full
`key = value` line (including multi-line `'''…'''`). See `parseExprs` in
`internal/scaffold/monorepo.go`.

Splice primitives: `parseExprs` → `sectionEnd` → `splice`. The engine reads and
rewrites the raw bytes with surgical insertions; it never round-trips through a
marshaller (which would lose comments).

## Family ↔ member drift coupling

Each `internal/coreassets/templates/monorepo/<family>.toml` template's `run`
strings must stay **byte-identical** to the member scaffolds' inline `run` bodies
(after `[vars]` substitution). Promotion relies on this equality check — a mismatch
means `PromoteMember` silently skips that task forever.

Enforced by drift tests:
- `TestNodeFamilyMatchesWebInline` (node family ↔ web member)
- `TestSwiftFamilyMatchesMemberInline` (swift family ↔ apple/swift-package/swift-cli)
- `TestGoFamilyMatchesMemberInline` (go family ↔ go-service member)

All in `internal/scaffold/monorepo_assets_test.go`. A drift-test failure means
**reconcile the two files**, never loosen the test.

## `tuist` placement

`tuist` is **apple-specific** and stays a member-level `[tools]` entry inside the
`apple` scaffold's `mise.toml` — it is **not** in the `swift` family's
`monorepo/swift.toml` contribution. The swift family contributes **no `[tools]`**
at all (swift and swift-format come from the active toolchain). Promotion of
apple's `test`/`fmt`/`lint` tasks works normally via `[vars] package_path = "Core"`;
the tuist-specific `build`/`generate`/`test:app`/`launch:macos` tasks have no
family template and stay inline permanently.

## Dependency-update gate (repo-global)

`EnsureRootMise` also injects a **repo-global Renovate deps gate** into the root
(every monorepo, from the first member): `ensureDepsGate` merges
`node`/`npm:renovate`/`jq` into the single root `[tools]` (deduped vs family pins,
never overwrites a user pin) and appends `[tasks.deps]` (advisory local dry-run —
never opens PRs, always exits 0) + `[tasks.check]`. `wireMonorepo` then drops
`renovate.json` + `scripts/deps-check.sh` at the repo root via
`EnsureDepsGateFiles` (skip-existing). **One ecosystem-agnostic gate covers all
members — it is NOT a family contribution**, so the swift-no-tools invariant holds
and Go/Swift stacks gain no node tooling of their own. There is **no per-member
deps task and no `dependabot.yml`** (the web scaffold's was removed). Source files
live at `internal/coreassets/templates/monorepo/{renovate.json,deps-check.sh}`.

**Gotcha:** the `npm:renovate` mise tool-backend key has a colon, so it must
serialize **quoted** (`"npm:renovate" = "latest"`) or the root mise.toml fails to
parse — `tomlKey` in `internal/scaffold/monorepo.go` handles this; the idempotency
compare still uses the decoded bare key.

See [[dev-workflow]] for the CI gate. Design: `docs/design/mise-monorepo.md`.
Implementation plan (historical): `docs/design/mise-monorepo-plan.md`.
