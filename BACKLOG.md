# SpecKit backlog

Running list of follow-ups, so nothing gets lost between sessions. **Open work is
ordered by priority** (P1 = next build → P6 = gated release); **completed work lives
in [Completed](#completed) at the bottom**. Status: ✅ done · 🔄 in progress ·
⬜ todo · 🔒 blocked.

---

## P1 · ✅ Shipped — GitHub Pillar 1 (PR gating)

Spec-honesty is now a non-bypassable required check. **Next build is P2.**

- ✅ **Web-scaffold quality tasks (prerequisite).** `fmt`/`fmt:check` (Oxfmt) + `lint`
  (Oxlint) mise tasks in the web scaffold, scoped to `app/`; `pnpm` pinned in `[tools]`.
  `.oxfmtrc.json` + `.oxlintrc.json` **mirror the Vite+ reference** (`tanstack-react-start-contacts`):
  Oxfmt with `tabWidth 4` / `printWidth 100` / `arrowParens avoid` / perfectionist import
  sort / **`sortTailwindcss`** / `sortPackageJson` / jsonc+`.vscode` overrides; Oxlint with
  type-aware rules + `jsPlugins` (perfectionist, prefer-let) + `import/extensions`. Deps:
  `oxlint oxfmt oxlint-tsgolint eslint-plugin-perfectionist eslint-plugin-prefer-let`.
  Green-on-arrival verified for real (`oxfmt --check` + `oxlint` + `tsgo` all exit 0 on a
  freshly rendered scaffold with the full dep set).
- ✅ **The gate Action + `ci.yml`.** Composite action `gate/action.yml`
  (`markmals/speckit/gate@v1`) + reusable workflow `.github/workflows/gate.yml`; the
  scaffold's `github/` subtree drops a thin-caller `.github/workflows/ci.yml` via the new
  `scaffold.RenderGitHub` (skip-existing) wired into `target add`. Two jobs: `quality`
  (mise tasks) + `verify` (the gate; runs the suite, so no double-run). Gate emits CI
  annotations via the new `specify gate … --format github` (mirrors `oxlint --format
  github`). **Deviation from the design doc, with rationale:** the CI sequence is `scan →
  gate firewall → verify → parity --gate`; **`gate generated`/`gate scope` are git hooks,
  not PR checks** — `verify` legitimately rewrites committed locks on green (a `generated`
  PR check would false-positive), and `scope` validates a single commit subject.
- ✅ **Branch-protection recipe** — documented `gh` ruleset fallback in
  [docs/ci-gating.md](docs/ci-gating.md) (required contexts `quality` + `verify / verify`,
  PR required, force-push blocked). `specify`-native provisioning rides on the gh-extension
  auth (P2).

**Needs a live PR to validate** (can't run an Actions runner locally): the mise/pnpm
toolchain setup + trust in CI, the firewall `--against` base-SHA diff, and the
`@v1`/`go install …@v1` references (dormant until the first release tag, P6).

**Fast-follows** (richer annotations — captured under P4): line-level firewall annotations
(`bindingsInContent` is regex-only, no line numbers today); and `--format github` on
`verify`/`parity` so **unjoinable scenarios, dangling bindings, and drifted specs** also
annotate to file:line — needs the report structs to carry the spec/test file path.

Design: [docs/design/github-integration.md](docs/design/github-integration.md).

---

## P2 · ✅ Shipped (mostly) — GitHub-native core + agent memory

The pivot's heart. Architecture: a portable spec-integrity core (engine works offline,
never needs GitHub for correctness) + a GitHub-native workflow shell; the engine
*projects* repo truth onto GitHub. Determinism line: specs/locks/parity/agent-memory
stay in the repo; defects/work/gating are **ephemeral** on GitHub. **The whole GitHub
surface lives in `internal/github`, imported only by `cmd/specify` — never by the engine,
so the offline guarantee holds structurally.**

- ✅ **GitHub foundation + gh-auth inheritance.** New `internal/github` package: a lean
  REST + GraphQL client that inherits `gh`'s token (`gh auth token`, env fallback) with
  zero plumbing and contains its own GraphQL queries (the gh-projects *approach*, our
  queries — no `go-gh` dep, no external extension). Repo auto-detected via `gh repo view`;
  **no config block.** `gh ≥ 2.94.0` + `op` pinned in `mise.toml` (repo + web scaffold).
  Fully unit-tested (httptest). **Deferred to release (P6):** the actual `gh extension
  install markmals/specify` distribution (running *as* `gh specify`) — the binary +
  commands are ready; only the release-tagged install path is dormant.
- ✅ **Pillar 2 — Issues as ephemeral defect intake.** `specify issues list|create|close`
  (confirmation-gated, `--json`); the `github/` scaffold subtree now drops the full
  per-target `.github/` (`PULL_REQUEST_TEMPLATE.md`, `ISSUE_TEMPLATE/defect.yml` stamping
  `type: Bug` + label fallback, `config.yml`, `CODEOWNERS` for `/features` `/specs`,
  `dependabot.yml`). No durable issue↔scenario link (GitHub cross-refs). Close-on-green is
  the discipline.
- ✅ **Pillar 3 — Projects board (Beads-informed).** `specify work ready|claim|move|discover`
  on the inlined Projects v2 GraphQL client (resolve project/fields/options, add item, set
  single-select, list items, atomic self-claim, `discovered-from` label + #N backlink).
  **"ready" is a Status column** (a `--column` flag, not a computed field). **Column names
  are flags** that default to the **confirmed** `APL-Innovation-Lab/projects/1` set —
  Backlog → **Ready** → In Progress → On Hold → Cancelled → Closed (actionable = **Ready**;
  "On Hold" is the blocked signal, skipped for free since `ready` lists only the actionable
  column). 🔄 **Remaining (minor):** `blocked-by`/epics (sub-issue) helpers exist in the
  client but have no dedicated `work` subcommand yet.
- ✅ **Agent memory (per-agent `memory/`).** `MemoryDir()` on the adapter →
  `.claude/memory/` · `.agents/memory/` · `.github/memory/`; `init` seeds `MEMORY.md`
  (skip-if-exists, so re-init never clobbers), wires loading (Claude `@import`;
  AGENTS.md/copilot directive), ships the `managing-memory` skill. Engine ignores it.
  Goldens updated; dogfooded → this repo's `.claude/memory/` + a root `CLAUDE.md`. Design:
  [docs/design/agent-memory.md](docs/design/agent-memory.md).
- ✅ **Deploy workflows (optional, none required).** `specify deploy add <kind> [target]`
  renders `deploy.yml` for `cloudflare-workers-ssr` / `cloudflare-workers-spa` / `railway` /
  `github-pages-spa` (templates use `[[ ]]` delims so `${{ }}` survives) and records the
  per-target manifest. Per-target (decided). (`init --deploy` sugar still TODO.)
- ✅ **Secrets via 1Password (`op`).** Manifest holds only `op://` references (validated;
  raw values rejected at `deploy add` + `scan`). `specify secrets sync` resolves via local
  `op` and pipes into `gh secret set` (CI) + `wrangler secret put` (cloudflare runtime, via
  stdin — never argv/log); railway runtime via argv (CLI limitation, noted). `--dry-run`
  prints the plan without resolving. **Default = `gh secret set` sync** (decided);
  runtime-load via `1password/load-secrets-action` remains a documented upgrade.
- ✅ **Branch-protection provisioning** (`specify protect`). Codifies the `docs/ci-gating.md`
  ruleset (require `quality` + `verify / verify`, require a PR, block force-push) via the
  GitHub API; re-runnable (updates an existing same-named ruleset in place).

**Resolved open decisions** (from the design doc): no `github` config block (auto-detect via
`gh`); GitHub commands un-namespaced; deploy is **per-target**; default secret mode is the
`gh secret set` **sync**; memory frontmatter is optional; **the Pillar 3 column set is
confirmed** (Backlog → Ready → In Progress → On Hold → Cancelled → Closed, actionable =
Ready) and baked in as `specify work`'s defaults.

---

## P3 · Scaffolding & stack coverage

- 🔄 **Web scaffold — flesh out** to the full default. Approach: prototype-first / resolve-by-running
  (build the real app green, then templatize), grounded in inspecting the reference apps (trove ·
  tangerine-dashboard · contacts main+convex). Engine decision: **plain Vite + oxc + Mise** (not
  vite-plus), confirmed with Mark — matches [scaffolds/web.md](docs/design/scaffolds/web.md).
  - ✅ **Slice 1 — default app, green-on-arrival (verified for real).** TanStack Start SSR +
    Router (virtual file routes, `app/routes.ts` → `routes.gen.ts` via `tsr generate`) + Query;
    React 19 + **React Compiler** (`@rolldown/plugin-babel` + `reactCompilerPreset()`); **Tailwind
    v4**; the **React-Aria + `cva` + Motion** foundation recipe (`app/components/foundation/`);
    Zod; `#/*` subpath imports; Mise tasks (dev/test/build/fmt:check/lint/typecheck + `routes`
    codegen, `mise trust` on install); the binding harness preserved. Fresh `specify target add web`
    → `verify` green + `fmt:check`/`lint`/`typecheck`/`build` all pass.
  - 🔄 **Slice 2 — `--data` layers.** Scaffold-manifest `data` variants + a `--data` flag
    (`scaffold.RenderData` overwrites shared base files like `router.tsx`; per-variant deps +
    codegen scripts phase-ordered after the base install).
    - ✅ **`--data convex` (the default) + `--data none`**, both green-on-arrival, verified for real.
      Convex is green offline via an **anonymous local deployment** —
      `CONVEX_AGENT_MODE=anonymous pnpm exec convex dev --once` generates `convex/_generated/`
      with no account/login; `pnpm-workspace.yaml dangerouslyAllowAllBuilds` clears pnpm 11's
      native-build gate. Client/provider wiring from the contacts@convex delta.
    - ⬜ **`--data drizzle` (D1)** — folds into the Cloudflare runtime slice below (D1 is reached
      via `cloudflare:workers`, so it needs that runtime).
  - ✅ **Slice 3 — runtime axis + Drizzle+D1.** A `--runtime` flag + a generalized variant
    mechanism (`scaffold.Variant`, `RenderVariant`, cross-axis `requiresRuntime`). **Cloudflare SSR
    is the default** (per web.md): `@cloudflare/vite-plugin` + `cloudflare()` + `wrangler.jsonc` +
    `wrangler types` codegen; `--runtime node` is the lighter (no-Cloudflare) variant. **`--data
    drizzle` is runtime-adaptive** (drizzle-orm@rc): a `Variant.RuntimeFiles` overlay ships the **D1**
    db module + `d1_databases` binding on `--runtime cloudflare` and the **`node:sqlite`** adapter
    (`drizzle-orm/node-sqlite`) on `--runtime node` — shared `schema`/`drizzle.config` +
    `drizzle-kit generate`. All of `{cloudflare,node}×{convex,drizzle,none}` verified green-on-arrival.
    ⬜ Deferred: the `--ssr`/`--server` spa/static modes (refs are all SSR).
  - ✅ **Slice 4 — `--with clerk`.** The `--with` Feature mechanism now carries deps + scripts
    (parallel to Variant), and features render LAST. Clerk feature (from tangerine): adds
    `@clerk/tanstack-react-start`, `app/start.ts` (`clerkMiddleware`), and wraps `root.tsx` with
    `ClerkProvider`. Green-on-arrival without live keys (typecheck + build pass; auth resolves at
    runtime).
  - ✅ **Slice 4b — `--with tiptap`.** A purely **additive** feature (the first non-provider one):
    adds `@tiptap/react`/`@tiptap/starter-kit`/`@tiptap/pm` (v3) + an `app/components/foundation/editor.tsx`
    `RichTextEditor` (StarterKit, `immediatelyRender: false` for SSR, cva/Tailwind-styled). Overwrites no
    shared file, so it composes with clerk and any data/runtime variant — render-tested + green-on-arrival
    verified for real (fmt:check/lint/typecheck/test/build + `specify verify` all pass on a fresh
    `target add web --with tiptap`).
  - ✅ **Slice 4c — provider composition seam + `--with posthog`.** Resolves the provider-stacking fork
    (Mark's call: use TanStack Router's `Wrap`). A new base `app/providers.tsx` (a Go template) is the
    single place client-side providers compose, via an accumulator (`tree = <X>{tree}</X>`) gated by
    `{{if .Features.<name>}}`; both the base and the convex `router.tsx` delegate their `Wrap` to
    `<Providers>`. So providers stack deterministically without fighting over a shared file — clerk
    (root.tsx) is orthogonal and composes for free. **posthog** is the first provider: `add: posthog-js`
    + a conditional `PostHogProvider` block (apiKey form, SSR-safe, env via `VITE_PUBLIC_POSTHOG_KEY`/`_HOST`);
    it carries no files (the wiring lives in the base seam). Also fixed: a base `pnpm-workspace.yaml`
    (`dangerouslyAllowAllBuilds`) so node+none can add build-script deps (posthog → core-js) — previously
    the one combo without it. Verified green-on-arrival for real across 6 combos: node×{none,convex}×{±posthog},
    cloudflare+convex (default), and node+none+**clerk+posthog** (both providers wired at once).
  - ✅ **Slice 4d — `--with email` (Resend + React Email).** Additive: ships `app/emails/welcome.tsx`
    (a React Email template) + `app/server/send-email.tsx` (a server-only Resend send helper, key via
    `process.env.RESEND_API_KEY`), overwriting no shared file. Deps `resend` + **`react-email` (v6)** +
    `@react-email/render` — note the standalone `@react-email/components` is EOL/deprecated; v6 consolidated
    the components into the `react-email` umbrella (clean, zero deprecated transitives). Green-on-arrival
    verified for real (fmt:check/lint/typecheck/test/build + `specify verify`). ⚠️ `resend` is a Node SDK;
    on the Cloudflare Workers runtime a user wiring the helper into a Worker may need node-compat (the
    helper is additive/unimported, so the scaffold itself stays green).
  - ⬜ Remaining `--with` add-ons: **sentry** (now unblocked: client provider rides the same
    `Wrap`/`providers.tsx` seam; note it also adds a vite plugin, which touches `vite.config.ts` — the
    *runtime* axis — so handle that overlap), then stripe (checkout route + server fns) / tanstack-db /
    electron.
  - ⬜ **Slice 5 — Varlock + `.vscode`** (the references don't wire Varlock — deferred/optional),
    and the web-development **pack refresh** to this stack.
- ⬜ **Library / non-app coverage** — add a product **`kind: app | library`** (relax the
  "actor is human" rule for libraries; `story`+`domain`+`error` apply, UI kinds don't) and the
  stacks **`swift-package`**, **`swift-cli`**, **`ts-lib`**, **`vscode-extension`**. Engine
  unchanged (joins scenario↔test regardless). Design:
  [library-products.md](docs/design/library-products.md). Roster is evidence-based (web ·
  website · apple · android · go-cli · node-cli · swift-package · swift-cli · ts-lib ·
  vscode-extension; dropped rust-cli/windows/linux/browser-extension).
- ⬜ **Per-stack scaffold builds** — **apple next** (exercises the `swift` report format + the
  `SpecTraits.swift` harness), then the rest one at a time, each gated on a tooling preview.
  node-cli already spec'd ([scaffolds/node-cli.md](docs/design/scaffolds/node-cli.md)).
- ⬜ **Feature-folder templates** (minor) — `NARRATIVE`/story/model/view-model/error templates
  under `.speckit/templates/feature/` so the commands scaffold faster (today they point at
  `specs/CONVENTIONS.md`, which works).

---

## P4 · Engine & workflow enhancements

- ✅ **Richer CI annotations (P1 fast-follow).** `--format github` now extends past the
  firewall: `verify` annotates unjoinable scenarios at their spec line + dangling bindings at
  the test line; `parity` annotates each non-conforming cell at its spec line; the firewall
  points at the exact `it(...)`/`@Test` line. `bindingsInContent` carries file+line,
  `specmodel.Scenario` carries its line, and `engine.SpecLocations` maps scenarios → spec
  file:line. The gate action runs `verify`/`parity --format github`. (Verified: a repro with
  an unjoinable scenario + a dangling binding emits the expected `::error file=…,line=…::`.)
- ⬜ **Scanner — multi-framework bindings.** Support the forms `CONVENTIONS.md` documents but
  the scanner doesn't yet read: MSTest `[TestProperty("scenario", …)]`, kotlin
  `@Tag("scenario:…")`, generic `// [scenario.id]` comment. (Today: Swift traits + Vitest
  titles only.)
- ⬜ **Product-rollup render** — `cover`/`parity` grouping + per-product verdict. `ProductTargets()`
  exists; lands with the multi-target slice.
- ⬜ **triaging-defects skill** — reframe around **GitHub Issues** (Pillar 2 supersedes the old
  `DEFECTS.md`-ledger blocker): triage an issue → scenario/regression test → close on green.
- ⬜ **Hooks (claude-pack overlay)** — `format-on-edit`, `spec-reconcile`, `stop-lint`,
  `notify-long-task`, `user-prompt-context` + codegen hooks (convex/openapi/tuist). `gate`
  already mechanizes the enforcement ones (firewall/generated/scope) — decide which hooks remain
  vs. folded into `gate`.
- ⬜ **codex/copilot review subagents** — equivalents to the claude-pack reviewers; their
  delegation models differ from Claude's dispatch (open question).
- ⬜ **VS Code extension** — codelens on `// SPEC:` (jump scenario ↔ bound test), drift gutter,
  a parity tree, run the gate, a board view. The developer-native complement to the CLI.

---

## P5 · Docs decisions & minor cleanups

- ✅ **Harness & usage guides (`docs/`).** Shipped the onboarding surface, linked from the README's
  new **Guides** section. Both axes, fact-checked against the `internal/project` adapters + the golden
  init manifests (the source of truth, kept in lockstep):
  - **Per harness** — [`docs/harnesses/{claude,codex,generic,copilot}.md`](docs/harnesses/): what `init`
    projects for each agent (exact paths for commands/skills/rules/subagents/`memory/` + the orientation
    file and its loading mechanism), how the `/speckit.*` commands are invoked there, and the per-agent
    differences (Claude: user-invocable skills + the 5 review subagents + `CLAUDE.md` native `@import`;
    Codex/Generic: the shared `.agents/` + `AGENTS.md` read-at-start projection, byte-identical to each
    other, no subagents; Copilot: dual `.github/agents`+`.prompts` command projection under `.github/`).
  - **With vs. without GitHub** — [`docs/usage/offline.md`](docs/usage/offline.md) (the engine alone:
    `scan`/`verify`/`lock`/`drift`/`cover`/`parity`/`gate` + git hooks, determinism line as the spine,
    no `gh`/network) and [`docs/usage/github.md`](docs/usage/github.md) (the optional shell: PR gate +
    `protect`, Issues/`taskstoissues`, the `work` board, `deploy`/`secrets` — every command marked
    optional). Reflects the shipped P2 surface.
- ⬜ **Decision — historical-doc vocab.** `FORK.md` / `FORK-PLAN.md` still use engine-key
  `platform` (~100× in FORK-PLAN) as dated planning records. Migrate to `target`, or leave as
  pinned artifacts? (Untouched for now.)
- ⬜ **Decision — `init --platforms` / `extension add` vs `target add --stack`.** The
  `features/0002-init` extension stories spec a surface that overlaps the shipped `target add` +
  `packs`. Reconcile, or relabel as a deferred Phase-4 design.
- ⬜ **Legacy whole-file templates.** `plan-template.md` / `tasks-template.md` /
  `checklist-template.md` still use the upstream `spec.md`/`plan.md`/`tasks.md` model + unrendered
  `__SPECKIT_COMMAND_*__` tokens — at odds with the feature-folder model.
- ⬜ **`render.go` PLATFORM header** — the cover/parity/gate state table prints a `PLATFORM`
  column; rename to `TARGET`.
- ⬜ **Discussions** (maybe) — spec RFCs before they're committed. Take it or leave it. **Not
  pursuing:** GitHub MCP toolset, GitHub Agentic Workflows.

---

## P6 · Release (outward — needs explicit go-ahead)

- ⬜ The first release (`v0.1.0` tag) activates brew + mise. On tag: goreleaser publishes
  archives and dispatches `specify-release` to the tap; the tap bumps + bottles. Checklist in
  `packaging/homebrew/README.md`. **Do not tag or push to the tap without confirmation.**

---

## Completed

- ✅ **Engine & config** — `.speckit/specs.json` (plain JSON: version/agent/paths/targets, verify
  wiring inline); the engine keys on **target** (`platform`→`target` rename through code, specs, and
  docs); `scan` validates the config. The full engine (scan/verify/lock/drift/cover/parity/gate) and
  `init` are implemented + tested.
- ✅ **Commands & skills** — all 9 `/speckit.*` commands reworked to the fork's reality (`.speckit/`,
  the `specify` engine, no scripts) synthesizing Workbench discipline + spec-kit rigor; 7 universal
  process/authoring skills + 13 platform-pack skills ported and wired to the commands; the
  feature-folder data model (`features/<NNNN>/…`, scenario sub-IDs, `// SPEC:` pointers) resolved as
  canonical.
- ✅ **Rules pack** — `code-quality` · `commit-discipline` · `spec-conventions` ·
  `enforcement-hierarchy` ported from Workbench to the fork's reality (`target`, `specify
  gate`/`verify`, the engine mechanizes the sync invariants) and projected by `init` into each
  agent's rules dir (`.claude/rules/` · `.agents/rules/` · `.github/rules/`), referenced from the
  orientation file (`@import` for Claude; a directive for AGENTS.md / copilot-instructions.md).
- ✅ **Subagents (claude-pack)** — `spec-reviewer` · `test-gap-finder` · `drift-hunter` ·
  `handoff-builder` · `visual-verifier` ported and projected into `.claude/agents/`.
- ✅ **Stack scaffolding (machinery + web)** — Part A (`config.AddTarget`/`Save`, the
  `internal/scaffold` text/template renderer, `specify target add … --stack`); Part B (the **web**
  scaffold green end-to-end: `target add web` → `pnpm add` → `mise run test` → `verify` green +
  locked); the create-sprinkles **resolve-by-running** alignment (phased `scripts`, `pnpm add` pins
  versions, no hardcoded `"latest"`); tsgo fix (`@typescript/native-preview` + `tsgo` typecheck task).
  web + node-cli stacks spec'd.
- ✅ **Distribution** — Homebrew from-source bottling (`specify.rb` + tap auto-bump + cross-repo
  dispatch); deploy gated on the first release (P6).
- ✅ **Docs & design** — repo-wide prose currency sweep (`platform`→`target`, de-upstreamed the
  inherited GitHub community files, removed dead upstream `.github/` config); `spec-driven.md` rewritten
  to the fork's model; design docs landed for **GitHub integration**, **agent memory**, **stack
  scaffolding**, and **library products**; mise task naming convention set to colons (`fmt:check`).
