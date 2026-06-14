# Handoff — SpecKit

**Snapshot: 2026-06-14.** A point-in-time handoff for the next agent. The durable
tracker is [BACKLOG.md](BACKLOG.md); update it as you go and delete this file when
its plan is stale.

SpecKit is a spec-driven framework: a Go binary (`specify`) that is both the engine
(scan/verify/lock/drift/cover/parity/gate) and the project bootstrapper.

---

## ← START HERE: replace the web foundation with **racket-ui** via the shadcn CLI

Mark's call: retire the web scaffold's hand-rolled `app/components/foundation/`
(React-Aria + cva recipe) and instead consume **racket-ui** — a shadcn/ui-compatible
registry rebuilt on React Aria Components + Tailwind v4 + cva + Tabler — through the
**official shadcn CLI**.

**The full recipe is already PROVEN GREEN** on a real scaffold (restructured to
shadcn-native + racket-ui installed → `fmt:check`/`lint`/`typecheck`/`test`/`build` +
`specify verify` all pass). **Read [`.claude/memory/rac-ui-shadcn.md`](.claude/memory/rac-ui-shadcn.md)
first** — it has the exact recipe + every gotcha. Two corrections to that note:

1. **The address is `markmals/racket-ui`** (the library was published under that name,
   NOT `markmals/rac-ui`). Local dir: `~/Developer/Libraries/racket-ui`. So the install
   is `pnpm dlx shadcn@latest add markmals/racket-ui/base markmals/racket-ui/button …`.

2. **⚠️ PRECONDITION — `registry.json` is missing from the published repo, so shadcn 404s.**
   `markmals/racket-ui` is public, but `registry.json` is **gitignored** there (untracked),
   so it isn't at the repo root. shadcn's GitHub registry *requires* `registry.json` at the
   root — `shadcn add markmals/racket-ui/…` currently fails ("raw.githubusercontent.com did
   not return a root registry.json file"). **Fix this first, in the racket-ui repo:**
   regenerate it (`mise run registry-sync`/`registry-build` there if stale), remove
   `registry.json` from `.gitignore`, commit + push it (it references the already-committed
   `registry/default/*` source files). **Verify it's fixed:**
   `pnpm dlx shadcn@latest list markmals/racket-ui` must succeed.

**Then templatize** (atomic — every combo must stay green; do NOT half-migrate):
apply the recipe across the base web scaffold + **both runtime** `vite.config.ts` files +
the **convex/drizzle** data variants + the **clerk/tiptap/email** features (all reference
`#/`) + `web_test.go`. Verify representative combos green-on-arrival against the **real**
GitHub registry — node+none, cloudflare+convex (default), and one feature stacked — then
`mise run ci`, PR, merge.

Notes: this supersedes Slice 4f's `lucide-react` (→ Tabler); **keep `tw-animate-css`**
(racket-ui's `globals.css` `@import`s it but doesn't declare it). racket-ui is still
original React-Aria + cva (+ Tabler) — no Catalyst/paid code, so the "no paid components"
rule below still holds.

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
- **racket-ui recipe ✅ recorded** (#20) — proven, gated on the publish fix above.

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
