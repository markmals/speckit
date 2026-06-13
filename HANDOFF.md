# Handoff — SpecKit

**Snapshot: 2026-06-13.** A point-in-time handoff for the next agent. The durable
tracker is [BACKLOG.md](BACKLOG.md); update it as you go and delete this file when
its plan is stale.

## Where things stand

SpecKit is a spec-driven framework: a Go binary (`specify`) that is both the engine
(scan/verify/lock/drift/cover/parity/gate) and the project bootstrapper. Recent work:

- **P1 ✅** — PR gating (the gate Action + `ci.yml`).
- **P2 ✅** — the entire GitHub-native core + agent memory (PR #3): `internal/github`
  (REST+GraphQL inheriting `gh auth token`), `specify issues`/`work`/`deploy`/
  `secrets`/`protect`, the Pillar-2 `.github/` scaffold files, deploy templates,
  per-agent `memory/` projection + the `managing-memory` skill.
- **`--repo`/`GH_REPO` fix** (PR #4) — fork-safe repo targeting.
- **P3 web scaffold ✅ through Slice 4** (PRs #5–#9): a green-on-arrival TanStack
  Start app across `{cloudflare,node} × {convex,drizzle,none}` + `--with clerk`.

- **P5 docs ✅** — the harness & usage guides (`docs/harnesses/{claude,codex,generic,copilot}.md`
  + `docs/usage/{offline,github}.md`), fact-checked against the `internal/project` adapters and
  linked from the README's new **Guides** section.

**Your job:** go down the remaining BACKLOG priorities in order — **finish P3 web
scaffold → P3 new stacks → P6 release.** Details per item below.

## Read these first

1. [BACKLOG.md](BACKLOG.md) — the live, prioritized tracker (✅/🔄/⬜ per item).
2. [CLAUDE.md](CLAUDE.md) → it `@import`s `.claude/memory/MEMORY.md` — the project
   memory holds the hard-won gotchas (engine boundaries, dev workflow, the web-scaffold
   recipes). **Read the memory files** — they'll save you the digging this session cost.
3. [docs/design/](docs/design/) — `github-integration.md`, `agent-memory.md`,
   `scaffolds/web.md`, `library-products.md` (the approved designs).
4. [CONTRIBUTING.md](CONTRIBUTING.md) + [specs/CONVENTIONS.md](specs/CONVENTIONS.md).

## How to work in this repo (conventions that bite if ignored)

- **`mise run ci` before every push** (build → vet → test + gofmt). It's the gate.
- **Golden init manifests**: any change to projected assets (`templates/{skills,rules,
  agents,commands,memory}/` or an adapter's paths) drifts the goldens —
  `go test ./internal/project -run TestInitGoldenTrees -update`, then eyeball the diff.
- **Scaffolds are prototype-first / resolve-by-running.** Build the *real* app green in
  a scratch dir (`pnpm add` pins versions; run `tsgo`/`oxlint`/`oxfmt`/`vitest`/`vite
  build` until green), THEN templatize. "Green-on-arrival" is *verified* by a fresh
  `specify target add <name>` → `specify verify` + the quality trio — **never asserted.**
- **Offline determinism line (core invariant):** `internal/engine` + `internal/specmodel`
  never import `internal/github` and never touch the network. All GitHub/network code
  lives in `internal/github`, imported only by `cmd/specify`. Keep it that way.
- **PR-per-change.** Branch off `main`, `mise run ci`, open + merge a PR.
  **Repo is `markmals/speckit`** — `gh` resolves the fork's parent (`github/spec-kit`)
  by default, so pass `-R markmals/speckit` (and `specify`'s GitHub commands honor
  `GH_REPO` / `--repo`).
- **Commit scopes are enforced** (`<scope>: <subject>`); valid scopes in
  [.claude/commit-scopes](.claude/commit-scopes).
- **No paid/closed component code in scaffolds** — never vendor Tailwind Plus / Catalyst
  (Trove's `foundation`); the web scaffold's `foundation` is original React-Aria + cva.
- **Keep README + docs/ + BACKLOG current** alongside usage-affecting changes.

## Toolchain gotchas (web scaffold — these cost real time to find)

- pnpm 11 hard-errors on native build scripts → `pnpm-workspace.yaml`
  `dangerouslyAllowAllBuilds: true` (esbuild/workerd). The explicit allowlist did NOT
  work in 11.5.3.
- **Convex green offline = anonymous local deployment:**
  `CONVEX_AGENT_MODE=anonymous pnpm exec convex dev --once` generates `convex/_generated/`
  with no account/login. Plain `convex codegen` fails ("No CONVEX_DEPLOYMENT").
- **React Compiler on Vite 8 / `@vitejs/plugin-react` v6:** a separate `@rolldown/plugin-babel`
  (DEFAULT import) pass running `reactCompilerPreset()`. v6 dropped the inline `babel` option.
- `cva` real package is **`cva@beta`** (plain `cva` = a 0.0.0 placeholder). Drizzle's
  `node:sqlite` adapter + `d1` are both only in **`drizzle-orm@rc`** (stable 0.45.2 lacks
  node-sqlite).
- A freshly rendered `mise.toml` is **untrusted** → scaffold install runs `mise trust` so
  `specify verify` (which shells `mise run`) is green-on-arrival.
- **E2E PITFALL (cost a repo-pollution incident):** do NOT `cd` inside a `$(...)` subshell
  in test harnesses — the `cd` is lost and a `specify target add` fires in the *current*
  repo. Always `cd "$dir/demo" && specify …` in the main shell, in a throwaway `mktemp -d`.

## The work, in priority order

### 1. P5 — Harness & usage docs  ✅ DONE
- **Shipped:** `docs/harnesses/{claude,codex,generic,copilot}.md` + `docs/usage/{offline,github}.md`,
  linked from the README's **Guides** section. Drafted + adversarially fact-checked against the
  `internal/project` adapters and the golden init manifests (the source of truth — kept in lockstep).
  Per-agent differences captured: Claude gets review subagents + a `CLAUDE.md` `@import` of rules+memory;
  Codex/Generic share the `.agents/` + `AGENTS.md` read-at-start projection (byte-identical, no subagents);
  Copilot dual-projects each command as a `.github/agents` chat-mode + a `.github/prompts` slash-prompt.

### 2. P3 — Finish the web scaffold  ← start here
- Remaining `--with` add-ons: stripe / email (Resend + React Email) / tiptap / tanstack-db /
  electron / sentry / posthog. The `--ssr`/`--server` **spa/static** modes. **Varlock + `.vscode`**.
  The **web-development pack refresh** to this stack.
- **Variant mechanism** (already built): `scaffold.Variant` (data + runtime axes) and
  `Feature` (`--with`) both carry `Files`/`Add`/`AddDev`/`Scripts`; `Variant.RuntimeFiles`
  is a runtime-keyed overlay (how drizzle ships D1 vs node:sqlite). `RenderVariant` /
  `RenderFeature`. Features render LAST (opt-ins win).
- Prototype-first; verify each new combination green-on-arrival.

### 3. P3 — New stack coverage
- **apple next** (Swift — exercises the `swift` report format + `SpecTraits.swift` harness),
  then the rest one at a time. Add a `kind: app | library` product +
  `swift-package`/`swift-cli`/`ts-lib`/`vscode-extension` stacks. Designs:
  [library-products.md](docs/design/library-products.md), [scaffolds/node-cli.md](docs/design/scaffolds/node-cli.md).

### 4. P6 — Release  ⚠️ needs Mark's explicit go-ahead to tag
- Wire the **`gh extension install` distribution** (a `gh-specify` build via goreleaser /
  `cli/gh-extension-precompile`) so `gh specify …` actually ships, then the first `v0.1.0`
  tag (activates brew + mise + the `@v1` Action refs). Checklist:
  `packaging/homebrew/README.md`. **Do NOT tag or push to the tap without confirmation.**

## Open validation gaps / follow-ups
- The GitHub **write** path (`specify work claim`/`move`) is unit-tested (httptest) and the
  *read* path + a no-op write were live-validated. With the `project` write scope now on the
  token, exercise `claim`/`move` against a throwaway board (`markmals/projects/2`), restoring
  state after. (Earlier no-op was correctly gated as a shared-board write — confirm before
  mutating.)
- The `gh specify` extension distribution is dormant until the release tag (P6).

## Don'ts
- Don't tag a release / push to the Homebrew tap without explicit go-ahead.
- Don't run scaffold e2e inside the speckit repo, and never `cd` in a `$()` subshell.
- Don't vendor Catalyst / paid component code into scaffolds.
- Don't let `internal/github` (or any network code) leak into `internal/engine`.

## This session's PRs (all merged into `markmals/speckit`)
#3 (P2 core + memory) · #4 (`--repo`) · #5 (web slice 1) · #6 (`--data`) ·
#7 (Cloudflare runtime + Drizzle+D1) · #8 (`--with clerk`) · #9 (drizzle runtime-adaptive).
