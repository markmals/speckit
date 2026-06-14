# Handoff — SpecKit

**Snapshot: 2026-06-14.** A point-in-time handoff for the next agent. The durable
tracker is [BACKLOG.md](BACKLOG.md); update it as you go and delete this file when
its plan is stale.

SpecKit is a spec-driven framework: a Go binary (`specify`) that is both the engine
(scan/verify/lock/drift/cover/parity/gate) and the project bootstrapper.

---

## ✅ JUST SHIPPED: web foundation → **racket-ui** via the shadcn CLI

The hand-rolled `app/components/foundation/` is retired; the web scaffold now consumes
**racket-ui** (`markmals/racket-ui`) — a shadcn/ui registry on React Aria + Tailwind v4 +
cva + Tabler — through the **official shadcn CLI**. Templatized shadcn-native (`@/*` →
member root via tsconfig `paths` + Vite 8 `resolve.tsconfigPaths`; `components/ui` + `lib`
at root; `app/` for routes/entry/`globals.css`; a new `components.json` with the
`@racket-ui` registries map; `#/`·`#db` → `@/`·`@/db`; `import/extensions: off`).
**Verified green-on-arrival for real across the full matrix** — `{cloudflare,node}` ×
`{convex,drizzle,none}` + all 5 features stacked. The real precondition (the handoff's
"commit `registry.json`" was insufficient — racket-ui's per-item `public/r/*.json`
distribution was gitignored, so the namespaced `@racket-ui/cva` dep 404'd) was fixed in
[racket-ui#1](https://github.com/markmals/racket-ui/pull/1) (merged). Recipe + gotchas:
[`.claude/memory/rac-ui-shadcn.md`](.claude/memory/rac-ui-shadcn.md).

## ← START HERE: finish the web scaffold (remaining `--with` + modes)

Next web slices, in order:

1. **`--with sentry`** — the client provider rides the same `Wrap`/`providers.tsx` seam
   (like posthog), BUT `@sentry/vite-plugin` also touches `vite.config.ts` — the **runtime**
   axis. So mirror the seam in BOTH `runtime/{cloudflare,node}/files/vite.config.ts` with a
   `{{if .Features.sentry}}` block (the provider seam fix, applied to the runtime axis).
2. **`--with tanstack-db`** (intersects the `--data` axis) and **`--with electron`** (a
   bigger shell change).
3. **`--ssr`/`--spa` modes** (trove proves SPA is needed — per-app) and **Varlock + `.vscode`**
   (Slice 5), plus the web-development **pack refresh** to this stack.

Prototype-first / resolve-by-running, then green-on-arrival across the affected combos
(see the web-scaffold + dev-workflow memories — esp. **verify combos SERIALLY**, the
type-aware `tsgolint` lint flakes under parallel-install contention).

---

## Where things stand

- **P1–P2 ✅** — PR gating + the entire GitHub-native core + agent memory.
- **P3 web scaffold ✅** — green-on-arrival TanStack Start across `{cloudflare,node} ×
  {convex,drizzle,none}`, with `--with` features **clerk · tiptap · posthog · email · stripe**
  (PRs #5–#15). The provider **`Wrap` seam** (`app/providers.tsx`) composes client providers.
  Foundation extras (lucide + tw-animate-css, #18) — lucide now superseded by racket-ui.
- **P5 docs ✅** — harness & usage guides (`docs/harnesses/*`, `docs/usage/*`), linked from
  the README's Guides section.
- **Tier 1 — trove engine adoption ✅** (#17). The engine can now `scan` + `verify` real
  Workbench-shaped repos. Added: the **`protocol`** kind; the filename rule accepts both
  id-tail *and* full-id stems; nested `### Scenario` parsing; a **Go + generic `// [scenario.id]`
  binding reader** (`.go` walked); the **`gotest`** report format (`go test -json`); and a
  per-target **`bindings: scoped`** mode (untagged tests out of scope; default stays `strict`).
  Validated end-to-end against `~/Developer/Projects/trove`.
- **go-service stack ✅** (#19) — a runnable Go HTTP daemon → `cmd/<name>` (verifies via the
  Tier-1 `gotest` format). Introduced **stack-aware member placement** (`Manifest.memberDir`):
  the first piece of **incremental member-add** (Mark's monorepo approach).
- **racket-ui foundation ✅ shipped** — the web scaffold's UI now comes from `markmals/racket-ui`
  via the shadcn CLI (see the JUST SHIPPED section above); racket-ui's per-item distribution
  published in [racket-ui#1](https://github.com/markmals/racket-ui/pull/1).

## How to work in this repo (conventions that bite if ignored)

- **`mise run ci` before every push** (build → vet → test + gofmt). The gate.
- **Golden init manifests**: any change to projected assets (`templates/{skills,rules,agents,
  commands,memory}/` or an adapter's paths) drifts them —
  `go test ./internal/project -run TestInitGoldenTrees -update`, then eyeball the diff.
- **Scaffolds are prototype-first / resolve-by-running.** Build the *real* app green in a
  throwaway `mktemp -d` (`pnpm add` pins versions; run the quality trio + `vite build` /
  `go test` until green), THEN templatize. Green-on-arrival is *verified* by a fresh
  `specify target add <name>` → `specify verify` + the quality checks — **never asserted.**
- **Offline determinism line (core invariant):** `internal/engine` + `internal/specmodel`
  never import `internal/github` and never touch the network. All GitHub/network code lives
  in `internal/github`, imported only by `cmd/specify`.
- **PR-per-change.** Branch off `main`, `mise run ci`, open + merge a PR. **Repo is
  `markmals/speckit`** — pass `-R markmals/speckit` to `gh` (it resolves the fork's parent
  by default). Mark's standing pref: **auto-merge green PRs** (build ×3 OSes + selftest);
  hold for review only when he says so (e.g. Tier 1 was held).
- **Commit scopes are enforced** (`<scope>: <subject>`); valid scopes in
  [.claude/commit-scopes](.claude/commit-scopes).
- **No paid/closed component code in scaffolds** — never vendor Tailwind Plus / Catalyst.
- **Keep README + docs/ + BACKLOG current** alongside usage-affecting changes.
- **Read the project memory** ([`.claude/memory/`](.claude/memory/)): engine-boundaries,
  dev-workflow, **web-scaffold** (feature/variant composition, the provider seam, toolchain
  gotchas), and **rac-ui-shadcn** (the recipe above).

## Toolchain gotchas (web scaffold — cost real time to find)

- pnpm 11 hard-errors on native build scripts → `pnpm-workspace.yaml`
  `dangerouslyAllowAllBuilds: true` (the base now ships one; the explicit allowlist did NOT
  work in 11.5.3).
- **Convex green offline = anonymous local deployment:** `CONVEX_AGENT_MODE=anonymous pnpm
  exec convex dev --once` generates `convex/_generated/` with no account.
- **React Compiler on Vite 8 / `@vitejs/plugin-react` v6:** a separate `@rolldown/plugin-babel`
  (DEFAULT import) pass running `reactCompilerPreset()` (v6 dropped the inline `babel` option).
- `cva` real package is **`cva@beta`**; drizzle's `node:sqlite`/`d1` adapters are **`drizzle-orm@rc`**.
- A freshly rendered `mise.toml` is **untrusted** → scaffold install runs `mise trust`.
- **tsgo / @typescript/native-preview removed `baseUrl`** (TS5102) — use bare `paths`.
- **E2E PITFALL (caused a repo-pollution incident):** do NOT `cd` inside a `$(...)` subshell
  in a harness — the `cd` is lost and `specify target add` fires in the *current* repo.
  Always `cd "$dir/demo" && specify …` in the main shell, in a throwaway `mktemp -d`.

## The rest of the backlog (after racket-ui)

1. **Finish the web scaffold** — remaining `--with`: **sentry** (client provider rides the
   `Wrap` seam, but `@sentry/vite-plugin` touches `vite.config.ts` — the *runtime* axis; mirror
   the seam fix there with `{{if .Features.sentry}}`), then tanstack-db / electron. Plus the
   `--ssr`/`spa` modes (trove proves SPA is needed — per-app) and **Varlock + `.vscode`** (Slice 5).
2. **Tier 2 monorepo (trove parity), incremental member-add** — make members compose into one
   workspace: `target add` maintains a root `pnpm-workspace.yaml` (+ catalogs), and go-service
   members share a **root `go.mod`** (today each is self-contained). Then **ts-lib** (`packages/<name>`)
   + the **ts-rest contract** package. Deferred Tier-1 follow-ups: protocol `x-spec` OpenAPI
   coverage reader; an `init`-into-existing path that authors `.speckit/specs.json`.
3. **New stack coverage** — **apple next** (exercises the `swift` report format + `SpecTraits.swift`),
   then `kind: library` products. Designs: [library-products.md](docs/design/library-products.md),
   [scaffolds/node-cli.md](docs/design/scaffolds/node-cli.md).
4. **P6 — Release** ⚠️ needs Mark's explicit go-ahead. Wire the `gh extension install`
   distribution + the first `v0.1.0` tag. **Do NOT tag or push to the tap without confirmation.**

## Don'ts

- Don't tag a release / push to the Homebrew tap without explicit go-ahead.
- Don't run scaffold e2e inside the speckit repo, and never `cd` in a `$()` subshell.
- Don't vendor Catalyst / paid component code into scaffolds.
- Don't let `internal/github` (or any network code) leak into `internal/engine`.
- Don't ship the racket-ui templatization until `registry.json` is committed to
  `markmals/racket-ui` and `shadcn list markmals/racket-ui` succeeds.

## This session's merged PRs (`markmals/speckit`)
#11 (P5 docs) · #12 (tiptap) · #13 (provider seam + posthog) · #14 (email) · #15 (stripe) ·
#16 (handoff refresh) · #17 (Tier 1 — trove engine adoption) · #18 (foundation extras) ·
#19 (go-service stack) · #20 (racket-ui recipe note).
