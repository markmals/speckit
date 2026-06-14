# Handoff — SpecKit

**Snapshot: 2026-06-14.** A point-in-time handoff for the next agent. The durable
tracker is [BACKLOG.md](BACKLOG.md); update it as you go and delete this file when
its plan is stale.

SpecKit is a spec-driven framework: a Go binary (`specify`) that is both the engine
(scan/verify/lock/drift/cover/parity/gate) and the project bootstrapper.

---

## ← START HERE: convert **trove** to SpecKit

Everything needed to *onboard* an existing repo now exists — the engine reads
trove's spec library clean (`specify scan` → clean), and **`specify target
register`** records an existing member as a target without scaffolding. The next
arc is the actual conversion of [`~/Developer/Projects/trove`](file:///Users/orion/Developer/Projects/trove)
(a Workbench-shaped monorepo). Its members:

| member | stack | format / bindings |
| --- | --- | --- |
| `cmd/troved`, `cmd/tangerined` | go-service | gotest · scoped |
| `cmd/trove-transcode` | go CLI | gotest · scoped |
| `apps/trove`, `apps/tangerine-dashboard` | web | junit · scoped |
| `packages/services`, `packages/tangerined-contract` | ts-lib | junit · scoped |

The conversion work, in order:

1. **Register every member** with its real wiring:
   `specify target register <name> --stack <s> --dir <path> [--command/--report/--source/--bindings …]`.
   Manifest stacks (web, go-service) seed the wiring; the others take explicit flags.
2. **The open engine question — multi-dir `source`.** A smoke-test (`register troved
   --source cmd/troved` → `verify`) showed trove's bound tests span **`cmd/troved`
   *and* `internal/`** (shared packages), and a target's `source` is a single dir
   today. Decide + implement: either a target `source` that spans multiple dirs, or a
   convention for shared-`internal/` bindings. This is the first real blocker.
3. **Reconcile scenario↔test ids** until each target verifies green (the smoke-test
   surfaced `scenario.library-scan.*` dangling/unjoinable under a too-narrow scope).
4. **Then:** the protocol `x-spec` coverage reader (verify trove's 17 protocol
   contracts against the OpenAPI — today they scan-clean + drift-track only) and the
   **product-rollup** render (per-product `cover`/`parity` — trove is multi-product:
   trove · tangerine).
5. **CI:** the gate action is dormant until the first release tag (P6); trove's CI
   needs the released action (or a pinned binary) to run `scan → gate → verify`.

Don't pollute the real trove while iterating — register writes only
`.speckit/specs.json` (+ any report files the test command emits); clean them up,
or work in a throwaway checkout.

---

## Where things stand

- **P1–P2 ✅** — PR gating + the entire GitHub-native core + agent memory.
- **P3 web scaffold ✅ (complete for now)** — green-on-arrival TanStack Start across
  `{cloudflare,node} × {convex,drizzle,none}`, with `--with` features **clerk · tiptap ·
  posthog · email · stripe**. UI comes from **racket-ui** (`markmals/racket-ui`) via the
  shadcn CLI — shadcn-native `@/*`→member-root layout, `components.json` with the
  `@racket-ui` registries map (racket-ui's per-item `public/r` distribution published in
  [racket-ui#1](https://github.com/markmals/racket-ui/pull/1)). **Slice 5** shipped a base
  `.vscode/` (oxc/tsgo/Tailwind-`cva` IntelliSense) + the web-development pack refresh;
  Varlock deferred. Recipe + gotchas: [`.claude/memory/rac-ui-shadcn.md`](.claude/memory/rac-ui-shadcn.md),
  [`.claude/memory/web-scaffold.md`](.claude/memory/web-scaffold.md).
- **go-service stack ✅ (trove-parity complete)** — members compose into ONE repo-root
  `go.mod` (**`sharedModule`**: each a `cmd/<name>` sharing `internal/`; module path from the
  git remote), plus three `--with` features matching troved's shape: **openapi**
  (contract-first via oapi-codegen strict-server + `x-spec`), **sqlite** (glebarez store +
  embedded migrations + flag/env config), **client** (bounded `http.Client` + `GetJSON`/
  `APIError`/URL-sanitization + a `fakeServer` httptest harness). `target add … --with
  openapi --with sqlite --with client` is the full troved shape, green on arrival.
- **`specify target register` ✅** — onboard an existing member as a target (no scaffolding);
  the enabler for the trove conversion above.
- **Tier 1 — trove engine adoption ✅** — `protocol` kind; filename rule (id-tail *and*
  full-id stems); nested `### Scenario` parsing; Go + generic `// [scenario.id]` binding
  reader; `gotest` format (`go test -json`); per-target `bindings: scoped` mode.
- **P5 docs ✅** — harness & usage guides (`docs/harnesses/*`, `docs/usage/*`).

## How to work in this repo (conventions that bite if ignored)

- **`mise run ci` before every push** (build → vet → test + gofmt). The gate.
- **Golden init manifests**: any change to projected assets (`templates/{skills,rules,agents,
  commands,memory}/` or an adapter's paths) drifts them —
  `go test ./internal/project -run TestInitGoldenTrees -update`, then eyeball the diff.
- **Scaffolds are prototype-first / resolve-by-running.** Build the *real* app green in a
  throwaway `mktemp -d` (`pnpm add`/`go get` resolve versions; run the quality checks +
  `vite build` / `go build` / `go test` until green), THEN templatize. Green-on-arrival is
  *verified* by a fresh `specify target add <name>` → `specify verify` + the quality
  checks — **never asserted.** **Verify combos SERIALLY** — the web type-aware `tsgolint`
  lint flakes (SIGSEGV / false `TS2307`) under parallel-install contention.
- **Offline determinism line (core invariant):** `internal/engine` + `internal/specmodel`
  never import `internal/github` and never touch the network. All GitHub/network code lives
  in `internal/github`, imported only by `cmd/specify`.
- **PR-per-change.** Branch off `main`, `mise run ci`, open + merge a PR. **The fork is
  `markmals/spec-kit`** — pass `-R markmals/spec-kit` to `gh` (bare `gh` resolves the
  upstream `github/spec-kit`). Mark's standing pref: **auto-merge green PRs** (build ×3
  OSes + selftest). **Stacked PRs:** merge in order; merging a parent with `--delete-branch`
  can *close* (not retarget) a child — retarget the child to `main` first, or merge the
  stack's tip (it carries all the commits) and close the intermediates.
- **Commit scopes are enforced** (`<scope>: <subject>`); valid scopes in
  [.claude/commit-scopes](.claude/commit-scopes).
- **No paid/closed component code in scaffolds** — never vendor Tailwind Plus / Catalyst.
- **Keep README + docs/ + BACKLOG current** alongside usage-affecting changes.
- **Read the project memory** ([`.claude/memory/`](.claude/memory/)): engine-boundaries,
  dev-workflow, web-scaffold, rac-ui-shadcn.

## Toolchain gotchas (cost real time to find)

**Web scaffold:**
- pnpm 11 hard-errors on native build scripts → `pnpm-workspace.yaml` `dangerouslyAllowAllBuilds: true`.
- **Convex green offline = anonymous local deployment:** `CONVEX_AGENT_MODE=anonymous pnpm
  exec convex dev --once`.
- **React Compiler on Vite 8 / `@vitejs/plugin-react` v6:** a separate `@rolldown/plugin-babel`
  (DEFAULT import) pass running `reactCompilerPreset()`.
- `@/*` resolves via tsconfig `paths` (**no `baseUrl`** — tsgo dropped it, TS5102) + Vite 8
  `resolve.tsconfigPaths`; `import/extensions: off` (rac-ui extensionless + codegen extensioned coexist).
- **racket-ui:** consuming a namespaced shadcn registry over GitHub needs its **per-item
  `public/r/*.json`** distribution committed (the aggregate `registry.json` alone only powers
  the *shorthand*); the consumer's `components.json` `registries` map points at the raw per-item URL.

**go-service scaffold:**
- Feature files that import a non-stdlib dep (sqlite→glebarez, openapi→oapi-codegen) **must be
  `.tmpl`** — else speckit's own `go build ./...` compiles them and fails on the missing import
  (the base `files/` package IS compiled by the suite, so it must stay valid stdlib Go).
- oapi-codegen runs via a `tool` directive (`go get -tool …`) + `go tool oapi-codegen`; the base
  `go mod tidy` moved to a **late phase** so feature codegen lands before tidy.
- Members import their own generated/internal pkgs by **full module path** (`Data.Module` =
  `resolveModulePath`); it's passed to templates *and* to the go.mod writer so they agree.

**Both:**
- A freshly rendered `mise.toml` is **untrusted** → scaffold install runs `mise trust`.
- **E2E PITFALL:** do NOT `cd` inside a `$(...)` subshell — the `cd` is lost and `target add`
  fires in the *current* repo. Always `cd "$dir" && specify …` in a throwaway `mktemp -d`.
- A scaffold **dotdir** template (`.vscode/`) whose name matches the repo-root `.gitignore` is
  silently dropped from commits (embed reads the working tree, so it builds locally but CI lacks
  it) — add a `!`-negation and confirm with `git check-ignore`.

## The rest of the backlog (after the trove conversion)

1. **Finish the web scaffold** — remaining `--with`: **sentry** (client provider rides the
   `Wrap` seam, but `@sentry/vite-plugin` touches `vite.config.ts` — mirror the seam on the
   *runtime* axis with `{{if .Features.sentry}}`), then **tanstack-db** / **electron**. Plus
   the `--ssr`/`--spa` modes (trove proves SPA is needed — per-app).
2. **Tier 2 monorepo (trove parity)** — the go-service shared `go.mod` ✅; still: a root
   `pnpm-workspace.yaml` (+ catalogs) maintained by `target add`, repo-root `internal/`
   sharing across go members, then the **ts-lib** (`packages/<name>`) + **ts-rest contract**
   scaffolds.
3. **New stack coverage** — **apple next** (exercises the `swift` report format +
   `SpecTraits.swift`), then `kind: library` products + a **go-cli** stack (trove-transcode's
   shape). Designs: [library-products.md](docs/design/library-products.md),
   [scaffolds/node-cli.md](docs/design/scaffolds/node-cli.md).
4. **P6 — Release** ⚠️ needs Mark's explicit go-ahead. Wire the `gh extension install`
   distribution + the first `v0.1.0` tag. **Do NOT tag or push to the tap without confirmation.**

## Don'ts

- Don't tag a release / push to the Homebrew tap without explicit go-ahead.
- Don't run scaffold e2e inside the speckit repo, and never `cd` in a `$()` subshell.
- Don't vendor Catalyst / paid component code into scaffolds.
- Don't let `internal/github` (or any network code) leak into `internal/engine`.
- Don't templatize a go-service feature file (importing a non-stdlib dep) as plain `.go` —
  it breaks speckit's own `go build`.

## This session's merged PRs (`markmals/spec-kit`)
#22 (web → racket-ui) · #28 (Slice 5 — `.vscode` + pack refresh) · #27 (go-service trove-parity:
root `go.mod` + `--with openapi`/`sqlite`/`client`, carrying #24·#25·#26) · #29 (`target register`).
Plus [racket-ui#1](https://github.com/markmals/racket-ui/pull/1) (the per-item registry distribution).
