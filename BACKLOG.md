# SpecKit backlog

Running list of follow-ups, so nothing gets lost between sessions. **Open work is
ordered by priority** (P1 = next build → P6 = gated release); **completed work lives
in [Completed](#completed) at the bottom**. Status: ✅ done · 🔄 in progress ·
⬜ todo · 🔒 blocked.

---

## P1 · Next build — GitHub Pillar 1 (PR gating)

The agreed next slice. Spec-honesty becomes a non-bypassable required check; the
demo is the firewall rejecting a test edited away from its spec.

- ⬜ **Web-scaffold quality tasks (prerequisite).** Add `fmt` / `fmt:check` (Oxfmt) and
  `lint` (Oxlint) mise tasks to `templates/scaffolds/web` so the `quality` CI job has
  standard task names to call (`test`/`fmt:check`/`lint`/`typecheck`).
- ⬜ **The gate Action + `ci.yml`.** Composite Action / reusable workflow
  (`markmals/speckit/gate@v1`) installs `specify`, runs `scan` → `verify <target>` →
  `parity --gate` → `gate firewall`/`generated`/`scope`, and emits **Checks-API
  annotations** (scenario → file:line). The scaffolded **`workflows/ci.yml`** is a thin
  caller with two parallel jobs: `quality` (the mise tasks) + `verify` (the gate; it
  already runs the test suite, so no double-run). Both required checks.
- ⬜ **Branch-protection recipe** — `specify` provisions the required-check ruleset via
  the GitHub API; ship a documented `gh` fallback.

Design: [docs/design/github-integration.md](docs/design/github-integration.md).

---

## P2 · GitHub-native core + agent memory

The pivot's heart. Architecture: a portable spec-integrity core (engine works offline,
never needs GitHub for correctness) + a GitHub-native workflow shell; the engine
*projects* repo truth onto GitHub. Determinism line: specs/locks/parity/agent-memory
stay in the repo; defects/work/gating are **ephemeral** on GitHub.

- ⬜ **`specify` becomes a `gh` extension** (foundational for Pillars 2–3). One Go binary:
  standalone `specify` (offline) **and** `gh specify …` (inherits `gh auth token` → zero
  token plumbing). Inline the Projects GraphQL (lifted from `NSExceptional/gh-projects`)
  in this repo; GitHub commands un-namespaced; likely **no config block** (auto-detect
  repo + linked project). **Pin `gh ≥ 2.94.0` via `mise.toml`.**
- ⬜ **Pillar 2 — Issues as ephemeral defect intake.** Scenario-canonical: defect Issue
  (org type Bug, via `defect.yml`) → fix adds/updates a scenario + regression test → close
  on `verify` green (lock = proof). **No durable issue↔scenario link** (rely on GitHub
  cross-refs). Per-target `.github/`: `ci.yml`, optional `deploy.yml`, PR template,
  `defect.yml`, `config.yml`, `CODEOWNERS` for `/features` `/specs`, stack `dependabot.yml`.
  Org Issue Types (Bug/Feature/Task + custom **Epic**); label fallback off-org.
- ⬜ **Pillar 3 — Projects as the work surface (Beads-informed, simplified).** Kanban;
  **"ready" is a Status column, not a computed field**. Epics = Epic-typed issue +
  sub-issues. Keep from Beads: `discovered-from:#N` provenance (label+backlink — the one
  GitHub gap), atomic claim, "land the plane" teardown; `blocked-by` as a visual signal
  only. Mirror Mark's `APL-Innovation-Lab/projects/1` columns (TBD — token lacks
  `read:project`).
- ⬜ **Agent memory (per-agent `memory/`).** Projected like skills: `.claude/memory/`,
  `.agents/memory/`, `.github/memory/`. `MEMORY.md` index loaded every session + topic
  files; committed. `init` wires loading (Claude `@import`; AGENTS.md/copilot directive);
  ship a `managing-memory` skill. Agent-owned (not `gate generated`-protected); the engine
  ignores it. Dogfood: this repo → `.claude/memory/`. Design:
  [docs/design/agent-memory.md](docs/design/agent-memory.md).
- ⬜ **Deploy workflows (optional, none required).** `deploy.yml` for `cloudflare-workers-ssr`,
  `cloudflare-workers-spa` (assets), `railway`, `github-pages-spa`. Chosen at
  `specify init --deploy`, addable later (`specify deploy add`); per-target vs project-level
  ergonomics open. (`CLOUDFLARE_ACCOUNT_ID` is committed in `wrangler.jsonc`, not a secret.)
- ⬜ **Secrets via 1Password (`op`).** 1Password is the single source of truth; repo holds
  only `op://` references, never values. `specify` resolves via local `op` and pushes to
  GitHub Actions secrets (`gh secret set`) + the platform store (`wrangler secret put` /
  `railway variables`), piping op→consumer (never echoed/logged). Optional upgrade:
  runtime-load via `OP_SERVICE_ACCOUNT_TOKEN` + `1password/load-secrets-action`. Pin `op`
  in `mise.toml`.

---

## P3 · Scaffolding & stack coverage

- ⬜ **Web scaffold — flesh out** to the full default: React Compiler, React Aria, Motion,
  Zod, TanStack Query/Table/Form/Hotkeys; the `app/` router structure; `--data convex|drizzle`;
  the SSR/server variants; `--with` features (clerk/stripe/…); Varlock + `.vscode`. Pack
  refresh to this stack. Stack approved in [scaffolds/web.md](docs/design/scaffolds/web.md).
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
