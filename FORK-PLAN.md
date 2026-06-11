# Fork Plan: speckit

A true fork of `github/spec-kit` that rewrites all executable code in Go (plus thin shell shims), adopts the Workbench (`../Workbench`) spec data model as core, adds the Mocha runtime engine (scan/verify/drift/cover/parity/ledger), and ships the curated multi-platform stacks as first-party extensions bundled with the core. No upstream contribution intended; divergence is the point.

This document is the seed artifact for agent-driven execution: decisions are resolved up front, phases have executable exit criteria, and open questions are marked `[NEEDS CLARIFICATION]` rather than silently defaulted.

## 1. Scope and posture

**What this fork takes from spec-kit:** the chassis. `init` and project bootstrapping, the per-agent command projection system, the extension/preset/override resolution stack, the catalog distribution model, the constitution, and the clarify/analyze/checklist command prompts.

**What it takes from Workbench:** the spec data model (CONVENTIONS.md becomes core law — dotted stable IDs, kind taxonomy, one-logical-thing-per-file, scenario sub-IDs, reverse pointers, deviation markers, `[NEEDS CLARIFICATION]` discipline), the process skills (TDD, verification-before-completion, systematic debugging, triaging-defects, DEFECTS.md), and the per-platform development skills.

**What it takes from Mocha:** the runtime engine (deterministic scan/verify/drift/cover/parity over the spec library), the acknowledgment-lockfile drift semantics, the run ledger and benchmark harness, the curated stack distro, and the GitHub workflow integration (gates, issues, parity check-runs).

**Fork posture:** pin the upstream commit at fork time and record it in `FORK.md` along with a module-by-module disposition (ported / rewritten / dropped). Never rebase. Cherry-pick upstream _prompt-level_ improvements (command markdown, template language) opportunistically — those are data, cheap to take. Never chase upstream's CLI behavior; after Phase 2 the implementations share a format lineage, not code.

**Licensing and identity:** spec-kit is MIT. Retain upstream's copyright notice in `LICENSE` and add your own. **The product name, binary, and slash namespace are deliberately retained — SpecKit, `specify`, `/speckit.*` (D1)** — the fork inherits that lineage rather than re-coining it; provenance rides on `FORK.md` and the retained license notice. What changes is the affiliation surface: replace the logo and visual identity, and drop `newsletters/`, `media/`, `.zenodo.json`, and all upstream branding. (Trade-off accepted: a near-identical name reads as a continuation of spec-kit, so any "not affiliated with GitHub" signal is carried by branding + `FORK.md`, not by the name.)

**Non-goals:** upstream contribution; byte-level conformance with the Python CLI; the full 30+ agent integration matrix; PowerShell script maintenance; hosting other people's extension catalogs (an upstream spec-kit roadmap feature — irrelevant here); live compatibility with upstream community extensions (the catalog is snapshotted as a design reference, not a live install target — see D6); the full platform matrix in v1 (web + apple ship first; the rest are additive post-v1 extensions — see D13); supporting spec-kit's monolithic `spec.md` format beyond a one-way importer.

## 2. Load-bearing decisions

### D1 — Name and binary

The product retains the **SpecKit** name (now spelling it without a space between Spec and Kit); binary is `specify`; slash namespace is `/speckit.*`. This is settled, not a placeholder: the fork deliberately inherits upstream's binary and command lineage rather than re-coining them (see §1 *Licensing and identity* for the affiliation trade-off).

### D2 — The runtime binary replaces the script layer

Spec-kit's runtime is bash + PowerShell scripts _because_ `specify` is absent after init — it's an installer, not a tool. This fork's binary is present at runtime (it's also the verification engine), so all script logic (`create-new-feature`, `check-prerequisites`, `setup-plan`, `setup-tasks`, `common`) moves into Go subcommands. Slash-command prompts call `specify feature new`, `specify prereqs --json`, etc.

This kills the bash/PowerShell dual-maintenance entirely and makes the agent-facing surface uniform: one binary, `--json` on every command, structured errors. Where shell remains: git hooks (2-line trampolines that `exec specify gate ...`). This satisfies the "Go or shell only" constraint with shell demoted to exec-trampolines.

### D3 — Core vs. first-party-extension boundary

**Core** (cannot be an extension because everything else depends on it): the spec data model and templates; the runtime engine (`scan`, `verify`, `drift`, `cover`, `parity`, `gate`, `lock`, `ledger`); init/integration/extension/preset management; the constitution and process command prompts (`/speckit.specify`, `/speckit.clarify`, `/speckit.analyze`, `/speckit.checklist`, `/speckit.apply`, `/speckit.reconcile`).

**First-party extensions, bundled and vendored in the release archive** (installable offline at init):

| Extension          | Contents                                                                                                                                      |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `platform-web`     | TanStack Start + Convex + Tailwind + React Aria stack: dev skill, scaffold, mise tasks, vitest verify adapter, CLAUDE.md fragment             |
| `platform-apple`   | UIKit/AppKit/SwiftUI(watch) + Observation + SwiftData: dev skill, Tuist scaffold, Swift Testing event-stream verify adapter + `SpecTraits.swift`, simulator-control skill |
| `platform-android` | Compose + Material 3 + Room: dev skill, Gradle scaffold, `kotlin.test` + Gradle JUnit-XML report adapter, emulator-control skill                  |
| `platform-windows` | WinUI 3 + MVVM Toolkit + EF Core: dev skill, scaffold, dotnet-junit verify adapter                                                            |
| `platform-linux`   | GTK 4 + Adwaita + Relm4 + Diesel: dev skill, scaffold, nextest-junit verify adapter                                                           |
| `platform-cli`     | Go (Cobra/Fang/Charm) — plus Rust and Node variants: dev skill, scaffold, verify adapter                                                      |
| `platform-website` | Astro + React islands                                                                                                                         |
| `backend-convex`   | Convex + Clerk: schema-as-protocol conventions, codegen tasks, contract generation from `convex function-spec`                                |
| `backend-openapi`  | Go + oapi-codegen server; per-platform generated-client conventions; contract conformance check wired into `specify verify`                   |
| `process-pack`     | TDD, verification-before-completion, systematic-debugging, triaging-defects skills (skills-mode assets)                                       |
| `github-pack`      | Action workflows, required-check wiring, spec→issues, parity check-run, PR-comment loop conventions, worktree helpers                         |
| `claude-pack`      | Claude Code overlay: subagents (drift-hunter, spec-reviewer, test-gap-finder, visual-verifier, handoff-builder), lifecycle hooks, `.mcp.json` |

**The key mechanism swap:** Workbench's `/setup` superset-then-prune model is deleted. `specify init --platforms web,apple,android --backend convex` is sugar for installing the corresponding bundled extensions. The curated distro _is_ the first-party catalog; promoting/demoting a stack is publishing/yanking an extension version. This is structurally better than pruning — additive, reversible, versioned per stack. **v1 bundles `platform-web` and `platform-apple` only** (D13); the remaining packs in the table are the post-v1 backlog — real, scoped, additive, but off the critical path until the engine has proven itself on the two platforms where the verify loop closes.

### D4 — Agent integration shortlist

Port **three agent families** — **Claude Code** (primary; skills mode), **Codex** (skills mode), and **GitHub Copilot** — each across its three surfaces: CLI, desktop GUI, and VS Code / editor extension. The projection adapter is per _family_, not per surface: within a family the surfaces share one config format (Claude's `.claude/` + `CLAUDE.md`; Codex's `AGENTS.md` + `.codex/`; Copilot's `.github/` instructions + `AGENTS.md`), so a single adapter's output serves that family's CLI, app, and extension alike — with surface-specific extras (e.g. VS Code prompt files) handled inside the adapter only where the surfaces actually diverge. `AGENTS.md` is therefore **not** a generic long-tail fallback (dropped — no other agents are in scope) but the shared substrate Codex and Copilot read natively, authored once. Cursor and every other upstream integration are dropped. Adding a surface or a family later is implementing/extending one Go interface (the projection adapter), documented as such.

### D5 — Command namespace unification

One namespace replaces both `/speckit.*` and `/sdd-*`: `/speckit.specify`, `/speckit.clarify`, `/speckit.plan`, `/speckit.tasks`, `/speckit.analyze`, `/speckit.checklist`, `/speckit.apply <spec> <platform>`, `/speckit.verify <platform>`, `/speckit.drift`, `/speckit.cover <spec>`, `/speckit.parity`, `/speckit.reconcile <platform>`, `/speckit.defect`, `/speckit.issues` (the taskstoissues descendant). `plan` and `tasks` survive but demoted per D9.

### D6 — Extension format posture

Adopt upstream's `extension.yml` schema as the _starting point_ for the fork's own manifest format — lift the structure, then own it; add fork fields under a namespaced `speckit:` key (verify adapters, scaffold entrypoints, platform identity, mise task fragments). **One dotdir: `.speckit/`** holds everything — runtime state (lock shards, ledger, platform manifest, config) and project-installed assets. There is no `.specify/` directory and no compatibility-shim layer. The upstream community catalog is snapshotted as a _design reference and idea mine_, not a live install target: community extensions are not promised to install, because their prompts hardcode `.specify/scripts/bash/*.sh` paths this fork deliberately doesn't have. Forking to diverge means the upstream format is ancestry, not a compat surface — paying for shims, dual dotdirs, and an eroding best-effort promise buys a benefit the posture itself disowns.

### D7 — Drift semantics: acknowledgment lock, not mtime

Workbench's CONVENTIONS mtime invariant is replaced wholesale (git doesn't preserve mtimes; CI clones and parallel worktrees make it vacuous). Drift state is a sharded lock — `.speckit/lock/<platform>/<spec-id>` records the content hash of the spec version last verified green plus per-scenario results — written only by `specify verify` on green, read by `specify drift` as hash-mismatch-or-missing. Sharded files, not one lockfile, so parallel worktree agents never merge-conflict in it. `specify lock` is the only writer; the path is covered by the generated-file gate.

### D8 — Enforcement lives in git and CI, not agent hooks

All honesty mechanics are `specify gate` subchecks callable from pre-commit and CI: scoped-commit subject validation, generated-file edit blocking, test-edit firewall (scenario-tagged test changed without its owning spec changing), lint-on-dirty-platforms, drift-on-PR. This makes enforcement agent-agnostic and stronger than Claude-only hooks. The `claude-pack` extension additionally wires the same checks into Claude Code lifecycle hooks for tighter in-session feedback — overlay, not substrate.

### D9 — Spec lifecycle: trunk-based library, disposable tasks

Specs live on main as the durable library (Workbench model); numbered feature branches are optional, not automatic (upstream already grew `--branch-numbering`; this fork defaults it off). `plan.md`/`tasks.md` survive as **disposable per-(spec × platform) execution artifacts** generated by `/speckit.apply` and deleted or archived after the lock goes green — resolving Workbench's "no plan documents" stance against spec-kit's load-bearing tasks.md. Durable state is the spec library + the lock + the ledger, never the task list. Worktree-per-agent is the parallel-execution model (`specify work start <spec> <platform>`).

### D10 — Windows support tier

Tier 1 as a _target platform_ (the WinUI pack). Tier 2 as a _host_: the Go binary is cross-compiled and tested on Windows CI, git-hook trampolines get `.cmd` variants, but no PowerShell logic is maintained anywhere. Doc toolchain: keep VitePress (build-time tooling is exempt from the Go-or-shell rule; it renders the spec library and isn't part of the product runtime).

### D11 — Deviation honesty is the engine's epistemic soft spot

The four-cell parity model (`conforming` / `declared-deviation` / `drifted` / `missing`) has one cell the engine cannot actually verify: `declared-deviation` is a human attestation (`// SPEC: <id> (deviates: <reason>)`), and a stale or self-serving marker reads identically to an honest one. The engine therefore treats `declared-deviation` as **human-attested, not machine-verified**: it surfaces every marker as a distinct review surface (a stale-deviation audit listing each deviation, its reason, and the last spec/impl change that touched it), and `specify parity --gate` treats a deviation cell as **"needs sign-off,"** never as green. The parity model gets its own spec — including the adversarial "the marker is lying" case — authored and pressure-tested in the Phase 1 spike before the real engine is built.

### D12 — The scenario-to-test join is a first-class, per-language conformance surface

The join (Gherkin `[scenario.<id>]` ↔ a real test in a platform's normalized junit output) is the load-bearing primitive: drift/cover/parity are only ever as honest as it is. Each language's join convention — comment tag, fn-name mangling (`scenario_items_list_empty`), or junit attribute — is documented as a spec and backed by a **per-language conformance fixture suite**. The engine **fails loudly, never silently**: a scenario with no joinable test, a test whose scenario ref is dangling, or a junit name the demangler can't parse is a hard `scan`/`verify` error, not a quiet "0 tests matched." This primitive is specced and conformance-tested before any verify adapter ships.

### D13 — v1 platform cut: web + apple

v1 ships **`platform-web` and `platform-apple` only** — the two platforms where the verification loop actually closes (browser visual verification via Chrome DevTools; simulator visual verification via `simctl`/`idb`). `platform-android`, `platform-windows`, `platform-linux`, `platform-cli`, and `platform-website` are the post-v1 backlog: real, scoped, and — because D3 makes every platform an additive extension — landable later with zero rework and zero lost vision. This keeps the v1 thesis-demo on the platforms where `spec → native → verified` is structurally complete, rather than shipping five packs whose verify story is unit-tests-only (windows/linux have no _behavioral_ UI verification until the parked a11y driver exists). Cutting them off the critical path is the single highest-leverage scope decision in the plan.

### D14 — Spec-first the rewrite; the Python CLI is a golden-output oracle, not the spec

The fork is built by its own process — the project is its own first user, fully. Before any Go is written for a capability, that capability is **specified** (stories + scenarios + golden-output acceptance criteria) in the spec library. The pinned Python CLI (`fork-base`) is used only to **capture golden outputs** — init trees, `--json` payloads, command output — as fixtures; its source is never read as the spec, so the Go implementation inherits the _behavior_, not Python's incidental structure. Per-capability loop: **spec it → capture goldens from the oracle → implement in Go from the spec → diff against goldens.** This makes Phase 0 a real "spec the retained CLI surface" effort (init+projection, check, version, self-upgrade, extension add/search/remove/--from/--dev, preset, catalog/manifest/install-state) _before_ Phase 2 implements it. Divergent behavior (dropped agents, no workflow engine, `.speckit/` dotdir, three agents × surfaces) is specced as the fork's own behavior; the oracle is consulted only for retained behavior. The golden fixtures — not the Python CLI — are the durable contract (§6).

### D15 — Scenario↔test join is source-bound; outcomes join by test identity

Verified in the Phase-1 spike: a test report need not carry the scenario ID. The binding is declared in **source** (a `.scenario(…)` trait, a `// [scenario.<id>]` comment) and the engine joins it to the report **outcome** by test identity (suite/class + test name). This is format-agnostic and retires test-name mangling. Per-platform authoring: **web** Vitest (the junit also happens to carry the id in `name`); **apple** Swift Testing with custom `.spec`/`.scenario` traits (`SpecTraits.swift`) and raw-identifier function names, outcomes from the `--event-stream-output-path` NDJSON (not the lossy `--xunit-output`); **android** `kotlin.test` (not raw JUnit) with a source binding, outcomes from the Gradle report. A test with no source binding, or a binding to an undeclared scenario, is a hard `scan`/`verify` error (D12).

## 3. Upstream disposition map (to be completed in Phase 0)

| Upstream asset                                                         | Fate                                                                                                                                                                                   |
| ---------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `src/specify_cli/commands/` (init, check, version, self-upgrade)       | Rewrite in Go                                                                                                                                                                          |
| `src/specify_cli/integrations/*` (30+)                                 | Rewrite 3 families (claude, codex, copilot) across their CLI/GUI/extension surfaces; `AGENTS.md` as the shared Codex/Copilot substrate; drop the rest                                  |
| `src/specify_cli/extensions.py`, `catalogs.py`, `integration_state.py` | Rewrite in Go (install state format owned by fork)                                                                                                                                     |
| `src/specify_cli/workflows/`, `authentication/`, `integration_runtime` | **Audit in Phase 0** — semantics unknown from outside; port minimally or drop                                                                                                          |
| `templates/` (spec/plan/tasks/agent-file)                              | Replace spec-template with Workbench per-kind templates; adapt plan/tasks per D9                                                                                                       |
| `scripts/bash/` + `scripts/powershell/`                                | Logic → Go subcommands; only shell left is git-hook trampolines (D2) — no `.specify/` compat shims (D6)                                                                                                                                    |
| `extensions/` catalogs + publishing guide                              | Fork as first-party catalog; community catalog snapshotted as a design reference only — not a live install target (D6)                                                                                                                    |
| `presets/`                                                             | Keep mechanism; audit shipped presets                                                                                                                                                  |
| `.github/` release pipeline                                            | Replace with goreleaser + template-package build in Go                                                                                                                                 |
| `docs/`, `spec-driven.md`                                              | Rewrite around fork identity; keep methodology essay as ancestry reference                                                                                                             |
| `tests/`                                                               | Mine as seed corpus for golden-tree tests                                                                                                                                              |
| Workbench `.claude/{skills,agents,hooks,commands,rules,templates}`     | Redistribute per D3/D8: skills → process-pack/platform packs; hooks → `specify gate` + claude-pack; commands → core prompts; rules → constitution + enforcement docs; templates → core |
| Workbench `specs/CONVENTIONS.md`                                       | Becomes core documentation _and_ the spec for the engine (D7 amendment applied)                                                                                                        |

## 4. Target repo layout

```
.
├── FORK.md                    # provenance: upstream commit, disposition map, divergence log
├── LICENSE                    # MIT, upstream notice retained + yours added
├── cmd/specify/               # Go CLI entrypoint
├── internal/
│   ├── engine/                # scan, verify, drift, cover, parity, lock, ledger, gate
│   ├── specmodel/             # frontmatter, IDs, kinds, scenarios, pointers (parser = CONVENTIONS.md mechanized)
│   ├── reports/               # JUnit-XML normalization + per-stack adapters
│   ├── project/               # init, template resolution stack, feature scaffolding
│   ├── integrations/          # claude, codex, copilot — per family, across CLI/GUI/extension (+ shared AGENTS.md emitter)
│   └── extensions/            # manifest, catalog, install state, priority/restore
├── templates/                 # core templates (per-kind specs, constitution, plan/tasks)
├── extensions/                # first-party packs (D3 table), each a valid extension repo-in-tree
├── catalog/                   # first-party catalog.json + snapshotted community catalog
├── prompts/                   # /speckit.* command markdown (agent-neutral; projected per integration)
├── docs/                      # VitePress
└── tests/                     # golden trees, engine corpus, conformance fixtures
```

## 5. Phases

Each phase is a worktree-parallelizable unit with executable exit criteria — with one structural exception: the **verify-adapter layer (Phases 3–4) is not parallelizable**. Its adapters bind to real toolchains on specific host OSes (Apple needs macOS + Xcode; each post-v1 platform adds another host), and their correctness is empirical, not generated — you learn the failure modes by running the real test runner and parsing its real (and churning) output, not by writing more Go. Throughput compresses the code around the adapters; it does not compress that loop. Plan the adapter work as a serial, host-bound, empirically-paced tail rather than a fan-out. The fork itself is built spec-first: Phase 0 writes specs for the CLI using the Workbench conventions, and engine features land with `[scenario.*]`-tagged tests — the project is its own first user.

### Phase 0 — Audit, decisions, seed specs (1–2 days)

Pin the fork commit. Walk `src/specify_cli` module-by-module and complete the §3 disposition map (resolving the `workflows/`/`authentication/` unknowns from source, not guesswork). Ratify D1–D15. Author the spec library for the **full retained CLI surface** (D14): `domain.specmodel`, the engine (`story.engine.scan/verify/drift/parity`), and the command surface (`story.init.*`, `check`, `version`, `self-upgrade`, `extension.*`, `preset.*`, catalog/manifest/install-state), with Gherkin scenarios and golden-output acceptance criteria. The pinned Python CLI is the **golden-output oracle** (capture its outputs as fixtures), never read as the spec.
**Exit:** `FORK.md` complete; decision log ratified; spec library for core commands passes `/speckit.analyze`-grade review; CI skeleton (Go build + lint) green on Linux/macOS/Windows.

### Phase 1 — Engine trust spike (throwaway; web + apple) (2–3 days, parallel with Phase 2)

Before committing to the engine's design, prove its two hardest claims on a deliberately disposable spike — hacky standalone code, not the real Go engine, kept only for its findings. Take one 3-scenario spec, hand-write a minimal TanStack Start impl and a minimal UIKit impl, and build _just enough_ to: (1) join those scenarios to real Vitest and Apple test output (D12), and (2) compute the four-cell parity matrix against a _deliberately lying_ `(deviates: …)` marker (D11). The question is binary: **is mechanized parity + the scenario join trustworthy enough to gate a PR on, or does the deviation cell need a human in the loop?** This runs concurrently with Phase 2 — the core binary doesn't depend on the engine being trustworthy — and it **gates Phase 3** (how the real engine is designed), not whether the chassis ships. The spike's only durable output is a findings memo and a list of join/parity failure modes folded back into the D11/D12 specs; the code is deleted or parked under `spikes/`, never promoted.
**Exit:** a one-page memo answering the gate question with evidence; D11 and D12 specs updated with the spike's discovered failure modes; throwaway code archived, not merged into `internal/engine`.

### Phase 2 — Core binary: init, templates, projection (~1 week)

Go skeleton (Cobra/Fang + Charm output, `--json` everywhere, structured errors). Port `init` (`--here/--force/--no-git/--platforms/--backend`), the four-layer template resolution stack with install-state and removal-restore, the three agent adapters (claude, codex, copilot) plus the shared `AGENTS.md` emitter, `check`, `version`, `self upgrade` (goreleaser-backed). Move all script logic into subcommands; generate git-hook trampolines. Golden-tree tests: `specify init` per agent produces the expected tree; for retained behaviors, diff against the Python CLI as oracle in a fixture job (oracle, not contract — it pins intent at fork time, then the fixtures own the truth).
**Exit:** fresh `specify init` yields a working project for all three agents across their CLI/GUI/extension surfaces; every command prompt invokes `specify`, zero runtime scripts beyond git-hook trampolines; golden trees green in CI on three OSes.

### Phase 3 — Spec engine (~1 week of Go + an empirical, host-bound adapter tail)

`specmodel` parser (frontmatter, dotted IDs, kinds, scenario sub-ID comments, reverse pointers incl. `deviates`/`manual`, per-language test tags). `specify scan` with lint (dangling `depends-on`, duplicate IDs, filename↔ID mismatch, scenarios missing sub-IDs, pointer health). `specify verify <platform>`: run the platform's mise task, normalize reports, and join to scenarios per D12. **v1 ships two adapters — Vitest junit and SwiftPM xunit/xcresult export** (the Gradle/dotnet/nextest/gotestsum adapters land with their packs post-v1, D13). The join handles both comment-tagged tests and fn-name mangling (e.g. Rust's `scenario_items_list_empty`, demangled by the scanner) and, per D12, hard-fails on any unjoinable scenario rather than silently matching zero tests. Sharded ack lock + `specify drift`; `specify cover`; `specify parity` with four cell states (conforming / declared-deviation / drifted / missing), the deviation cell gated for human sign-off per D11 (with a stale-deviation audit). `specify gate` with the test-edit firewall and scoped-commit check. **The adapter half is empirical and host-bound** — budget for it as a serial tail, not part of the Go week (§5 intro, §7).
**Exit:** against the web + apple sample: edit a spec → `specify drift` red → `/speckit.apply` → `specify verify` green → lock written → `specify drift` clean. The firewall demonstrably blocks a test edit without a spec edit. A planted lying `(deviates: …)` marker is caught by the stale-deviation audit and parity refuses to mark it green (D11); a scenario with no joinable test hard-fails `scan` (D12). All engine behavior covered by tagged tests against fixture repos.

### Phase 4 — Extension manager + first-party packs (web + apple) (~1 week)

Port `extension add/search/remove/--from/--dev` and `preset` with priority/restore semantics; vendored first-party catalog resolvable offline; snapshot the upstream community catalog as a design reference only (no compat shims, D6). Author the **v1 packs — `platform-web`, `platform-apple`, `backend-convex`, `process-pack`, `claude-pack`** (D13); each platform pack supplies its scaffold, dev skill, CLAUDE.md/AGENTS.md fragment, mise tasks, and a verify-adapter manifest the engine consumes (the spike's findings now hardened into real adapters). `specify init --platforms` sugar lands here. The post-v1 packs (android/windows/linux/cli/website, backend-openapi) are the same machinery applied again — no new mechanism, off the critical path.
**Exit:** `specify init --platforms web,apple --backend convex` scaffolds two buildable apps with wired verify adapters; adding then removing a pack round-trips — install/remove restores prior command versions correctly; the offline catalog resolves with no network.

### Phase 5 — GitHub pack + example repo (~1 week)

The Action: scan/verify/drift/parity as required status checks, parity matrix posted as a check-run summary (deviation cells shown as needs-sign-off, not green, per D11), `/speckit.issues` (spec/scenario → GitHub issues), PR-comment-to-agent loop conventions, `specify work start <spec> <platform>` worktree helpers. Generate the **companion example repo** (the contacts app, web + apple) from the fork and keep it green in CI — it is simultaneously the demo, the fork's integration test, and the substrate for the gates. **This repo is the engine's trust gate**: Phase 6's public release and bench publishing don't begin until it has stayed green through a real multi-spec, multi-platform lifecycle.
**Exit:** an example-repo PR shows the full lifecycle: spec edit turns drift check red on the PR, apply+verify on the implicated platforms turns it green; parity matrix renders in the check run, with a declared deviation correctly shown as needs-sign-off rather than green (D11).

### Phase 6 — Ledger now; bench + public release once earned (ongoing)

The ledger lands now because it's cheap and immediately useful: `specify apply` appends JSONL records (spec, platform, attempts, per-iteration scenario results, wall time, tokens where available, fail-first-observed flag). **Everything downstream of the ledger is gated on the example repo having earned trust** (Phase 5): `specify bench` (replays a spec set against candidate stacks → comparison table — the Mocha curation harness as a dogfooding byproduct; the distro's picks get receipts), the public docs site, and the release train (goreleaser → GitHub Releases → mise registry, then brew/winget). Publishing bench "receipts" or a `mise use -g speckit` storefront before the parity engine is trusted on a real project is shipping the storefront before the product — so it waits.
**Exit (gated on Phase 5 staying green through a real lifecycle):** first published bench table for ≥2 stacks over the example spec set; `mise use -g speckit` installs a working binary on all three host OSes (Windows host per D10 tier-2). Until that gate is met, this phase ships only the ledger.

**Parked (deliberately not in scope):** the accessibility-tree behavioral driver for the desktop trio (the genuinely unsolved frontier — schedule as its own project once the engine exists); visual-verifier automation beyond the claude-pack subagent; self-hosted third-party catalogs; importer for spec-kit monolithic specs (write when a real repo needs it).

## 6. Verification strategy for the fork itself

Per **D14** the rewrite is spec-first: specs + golden fixtures are the contract, and the pinned Python CLI only seeds the goldens (capture, never transcribe). Four layers. (1) **Tagged unit/behavior tests** per the fork's own conventions — the engine's spec library is real, not decorative. (2) **Golden-tree fixtures** for init/extension/preset operations, the durable contract once the Python oracle is retired. (3) **Oracle diffs** against the pinned-commit Python CLI for intentionally-retained behavior, Phase 2 only — they pin fork-time intent, then fixtures take over (a true fork must not let upstream's implementation remain the long-term spec). (4) **The example repo in CI** as the end-to-end integration test; if it rots, the fork rots visibly.

## 7. Risks and standing costs

The rewrite is the cheap part; these aren't. **The parallelism cliff**: the value of the whole system lives in the verify-adapter layer, and that layer is exactly where worktree-parallelism stops working — each adapter binds to a real toolchain on a specific host OS (Apple needs macOS + Xcode; Windows needs Windows), and its correctness is learned empirically by running the real test runner, not generated. Plan the adapter work as a serial, host-bound, empirically-paced tail — and note it's the structural reason v1 cuts to two platforms (D13). **xcresult/xunit churn**: Apple's test-report surface moves with each Xcode; isolate behind the report-adapter interface and expect yearly maintenance. **Deviation honesty (D11)**: `declared-deviation` is the one parity cell the engine can't verify; a stale or self-serving `(deviates: …)` marker is indistinguishable from an honest one, so it's gated for sign-off and surfaced as its own audit — but the residual risk is a project that rubber-stamps deviations until parity means nothing. **Join fragility (D12)**: the scenario-to-test join is the load-bearing primitive; a renamed test, a changed scenario ID, or a truncated junit name silently degrades "the right tests pass" to "some tests pass" unless the engine fails loudly on every unjoinable scenario — which is why D12 makes that a hard error, not a warning. **Behavioral UI verification off web/mobile**: report parsing is fine everywhere, but _behavioral_ UI verification on Windows/Linux desktop is unsolved territory — when those packs ship post-v1 they carry unit/behavior-test verification only until the parked a11y driver exists; their READMEs must say so rather than imply parity with web/apple. **Agent-format churn**: Claude Code, Codex, and Copilot each move their command/skill formats — and can do so independently across their CLI, GUI, and editor-extension surfaces. The hedge is per-family adapters (three, not nine surface-specific ones), since each family shares one config format across its own surfaces; the standing cost is tracking where a family's editor surface diverges from its CLI. **No community-extension compat, by choice (D6)**: the upstream catalog is reference-only — the risk here isn't maintenance (there's none) but expectation-setting, so document loudly that community extensions are not an install target. **Curation load**: the distro's ongoing cost is judgment — watching stacks, promoting/demoting packs, re-running `specify bench` per model generation — which is the job you signed up for and the part no agent throughput reduces.

## 8. Kickoff sequence (first two days, concretely)

Fork, pin, strip branding, land `FORK.md` with the §3 map stubbed. Confirm the now-resolved decisions (D1 naming; D4's three-agent, multi-surface shortlist) and the new scope decisions D11–D15. Write the Phase-0 spec library for `specify init`, `specify scan`, `specify verify`, `specify drift` — Gherkin scenarios first, lifted from this plan's exit criteria — plus the D11 parity-model spec and the D12 join spec, since the Phase 1 spike exists to pressure-test both. Stand up the Go module, CI matrix, and goreleaser config. Then run two tracks in parallel: the **Phase 1 engine spike** (throwaway web+apple, answering the parity/join gate question) and the **Phase 2 core binary** (the proven chassis, which doesn't depend on the spike's outcome) — the spike's findings gate how the Phase 3 engine is built, not whether the chassis ships. `specmodel` is the package the core binary and the real engine share, so land it first by itself; the throwaway spike can parse scenarios however it likes. From there the loop is the one you already run: spec → apply → verify → lock — on the tool that implements it.

# Mocha

Mocha is a spec-driven toolkit for building end-user applications across native platforms.

It gives each app one shared product model, backend, design system, test suite, and deployment workflow while preserving native implementation on every platform. Web apps use TypeScript and React/TanStack. Apple apps use Swift and UIKit/AppKit. Android apps use Kotlin and Jetpack Compose. Windows apps use C#, XAML, and WinUI. Linux apps use Rust, GTK, and Adwaita.

Mocha is built for agent-assisted development. Product specs, data models, design tokens, workflows, and acceptance tests are first-class project artifacts, so agents can generate, update, test, and explain changes across the full application without flattening every platform into a single runtime.

A strong agent-native project has:

- canonical specs
- durable architectural notes
- reverse pointers from code to specs
- explicit acceptance criteria
- executable verification
- deterministic formatting/linting
- scoped task commands
- generated context summaries
- drift detection
- hooks that prevent the agent from lying to itself

## Features

- App specification
  - business requirements
  - feature specs
  - use cases
  - user stories
  - domain models
  - permissions
  - workflows
  - routes/screens
  - acceptance tests
- Platforms
  - Web apps — TypeScript, React, TanStack (SSR/SPA)
  - Websites — TypeScript, React, Astro
  - iOS — Swift, UIKit
  - iPadOS — Swift, UIKit
  - macOS — Swift, AppKit
  - tvOS — Swift, UIKit
  - watchOS — Swift, SwiftUI
  - Android — Kotlin, Jetpack Compose
  - Windows — C#, XAML, WinUI
  - Linux — Rust, GTK, Adwaita
  - TUI — Go, Bubble Tea
  - RESTful server — Go, `oapi-codegen`
  - Realtime server — TypeScript, Convex
- Data platform
  - database
  - file storage
  - auth integration
  - search
  - workflows
  - cron jobs
  - background jobs
  - realtime
  - logging
  - local-first sync
- Design system
  - W3C design tokens
  - CSS variables and utility classes for web
  - Xcode color/image/font assets for Apple
  - Android Material tokens
  - WinUI resource dictionaries
  - GTK/libadwaita style resources
  - generated documentation
  - token-aware linting
- Tooling
  - JavaScript runtime
  - TypeScript type checker
  - Build system
    - Rolldown + Oxc
    - Swift Build
    - Gradle
    - MSBuild
    - Cargo
    - `goc`
  - Scaffolding
  - Package manager
    - npm
    - Swift Package Manager
    - Gradle
    - NuGet
    - Cargo
    - `go get`
  - Formatter
    - Oxfmt
    - `swift-format`
    - ktfmt
    - dotnet format
    - rustfmt
    - `go fmt`
  - Linter
    - Oxlint
    - `swift-format`
    - ktlint
    - StyleCop
    - Clippy
    - `go vet`
  - Test runner
    - Vitest
    - Swift Testing
    - `kotlin.test`
    - MSTest
    - Cargo Test
    - `go test`
- Testing
  - Behavior tests
  - Unit tests
  - Snapshot tests
  - Integration tests
  - End-to-end tests
- Virtualization
  - Chrome DevTools
  - iOS, tvOS, watchOS simulator
  - Android emulator
  - macOS container
  - Windows container
  - Linux container
- Deployment
  - Backend to Railway
  - Backend to Convex
  - Backend to Cloudflare Workers
  - Web to Cloudflare Workers
  - Apple platform to TestFlight/App Store
  - macOS to notarized app
  - Android to Internal Testing/Play Store
  - Windows to Windows Store
  - Linux to Flatpak
  - CLI to Homebrew, WinGet, apt, pacman, dnf, pkg, nix, npm

## The Distro

Mocha contains within it a distro manifest for the technical stack of an agent-native development project. The ongoing work is judgment: watch the ecosystem, promote/demote choices, document _why_ each pick is agent-legible, maintain a `setup` command for pruning as toolchains churn.

## Universal Hygiene Tooling

The value for this is that agents need a deterministic cleanup pass after every edit. They need uniform diagnostics, stable autofixes, and low-noise output. A formatter, linter, and typechecker become part of the agent’s proprioception: it tells the agent what it just broke and what can be fixed mechanically.

The best version is probably:

**one command surface over many native formatters, linters, typecheckers, and static analyzers, with normalized machine-readable diagnostics.**

This fits the multiplatform worldview for Swift, Kotlin, TypeScript, Rust, Go, and C#.

### Semantic Lint as Policy

The agent-era version of the "universal linter" instinct: lint rules generated _from specs_ and enforced mechanically — reverse-pointer health, naming-law compliance, scenario-tag coverage. Worth naming as its own pattern: spec-aware static checks that encode project policy for agents.

## Agent-Optimized Toolchain

A toolchain and environment manager with agent-native DX, optimizing for:

- instant project understanding
- typed task graph introspection
- structured error output
- dependency graph queries
- test impact analysis
- file ownership/spec ownership lookup
- safe sandboxed command execution
- automatic context condensation
- reproducible env loading
- built-in OpenAPI/schema/spec awareness

In other words, the toolchain should make the agent’s observe → edit → run → diagnose → repair loop tighter.

The toolchain should be a single-install, one-stop-shop. You don't need to go collect formatters and linters and bundlers and config files and toolchains. It should mimic the ease of polyglot development using Mise, but with a deeper integration for our blessed stacks and languages. One CLI the agent can use to maintain an entire project for its development needs.

## Verification Infrastructure

Agents can generate code quickly. The scarce thing is trustworthy closure. So tools that convert product intent into executable checks are more valuable than tools that merely make implementation nicer.

Useful directions:

- Gherkin-to-test scaffolding
- spec coverage maps
- visual regression tied to story IDs
- accessibility checks tied to component specs
- API contract conformance
- “which specs does this diff affect?”
- drift reports across native platform implementations
- CI summaries written for agents, not humans

## Agent Memory and Context Packaging

Claude Code skills and repository-level guidance are now a serious DX surface. Anthropic’s docs describe skills as reusable instructions/procedures Claude can invoke, and hooks as deterministic lifecycle controls rather than model-dependent behavior.

That means the new "framework design" might be:

- good `CLAUDE.md`/`AGENTS.md` structure/conventions
- good skill APIs
- good subagent boundaries
- good promptable command contracts
- good handoff files
- good context compaction
- good machine-readable project maps

This is very close to what framework authors used to do: define primitives, lifecycle, conventions, and extension points. The difference is that the consumer is now an agent, not primarily a human programmer.

## Mocha CLI — Intent-Only Commands

Commands like `mocha spec drift`, `mocha spec cover`, `mocha spec parity`, scenario-to-test joins. This is a layer where the problem (keeping N native implementations of one spec honest) barely existed two years ago, because agents are what just recently made N-platform native development tractable. The connective tissue is some of the only genuinely novel software left in the system once frameworks are curated rather than created.

## Framework Curation Benchmarks

The missing feedback loop: the claim that UIKit + Observation beats SwiftUI for native look and feel when implemented by agents, or that Relm4 beats raw GTK for agent comprehension and understanding, is currently vibes. Build a small harness: same spec, `mocha spec apply <spec-id>` against candidate stacks, measure pass rate, iterations-to-green, token cost, recovery behavior after injected failures. Nobody is publishing "agent throughput per framework" data. It turns taste into evidence and makes the distro self-justifying.

# Workbench

![Workbench hero image](./docs/public/workbench-hero.png)

> A Claude-Native Spec-Driven Development Template

A GitHub template for building **spec-driven multiplatform apps with Claude Code** as your primary collaborator. Specs are the source of truth; every platform implements them natively. There is no shared code — reconciliation across platforms happens through agent-mediated regeneration, not through a shared library.

This template assumes you will work with [Claude Code](https://claude.com/claude-code) every day. Every convention, file, and workflow in here is shaped by that assumption.

## Quick start

1. Click **"Use this template"** on GitHub to create your repo.
2. Clone it locally and open it in your editor.
3. **Run [`/setup`](.claude/commands/setup.md) in Claude.** This template ships as the _superset_ of every platform the stack supports (see [`STACK.md`](specs/STACK.md)). `/setup` asks which platforms and backend you're actually shipping and prunes the skills, hooks, permissions, and docs for everything else — turning the superset into just your project.
4. Customize the seed content:
   - [`specs/ARCHITECTURE.md`](specs/ARCHITECTURE.md) — fill in the `[NEEDS CLARIFICATION]` product overview and out-of-scope sections.
   - [`specs/DESIGN_SYSTEM.md`](specs/DESIGN_SYSTEM.md) — adjust tokens once branding is settled.
   - [`.env.schema`](.env.schema) — declare your environment variables (Varlock's typed contract).
   - `docs/index.md`, `docs/.vitepress/config.ts`, `docs/.vitepress/theme/components/Hero.vue` — set the project title.
   - `docs/public/` — replace `workbench-hero.png`, `workbench-icon.png`, `favicon.svg` with your own brand art.
5. Author your first feature: invoke the `brainstorming-feature` skill in Claude. It populates `features/0001-<your-slug>/`.
6. Scaffold your reference platform under `apps/<platform>/`. Add the per-platform `CLAUDE.md` and `mise.toml`. (See [`CLAUDE.md`](CLAUDE.md) for the recommended layout.)
7. Implement the feature on the reference platform with the `implementing-a-spec` skill, then mirror to other platforms via `/sdd-apply <spec-id> <platform>`.

## What "Claude-native" means

Lots of templates can be _used with_ an AI assistant. This one is _designed for_ one. Concretely:

- **The spec is the contract; the agent is the implementer.** Specs in [`specs/`](specs/) and `features/<NNNN>-<slug>/` describe behavior in a form Claude can read, reason about, and translate into native code on each platform. Implementations carry `// SPEC: <id>` reverse pointers so Claude can trace from a line of code back to the spec it came from — and detect drift in the other direction.
- **Workflows live as skills, not in your head.** Recurring procedures (writing a story, implementing a spec, debugging, verifying before claiming done) are encoded in [`.claude/skills/`](.claude/skills/) so that any session — yours, a teammate's, a fresh agent — picks up the same discipline.
- **Cross-cutting checks live as subagents.** Audits like drift detection, spec review, test-coverage gaps, and visual verification run as isolated subagents in [`.claude/agents/`](.claude/agents/) so they don't pollute the main conversation context.
- **Repetitive judgment is automated as hooks.** Formatting on edit (per language), blocking edits to generated files, regenerating Convex / Tuist projects, reminding you to regenerate OpenAPI clients when the contract changes, linting on stop, surfacing reconciliation reminders when a spec changes — all in [`.claude/hooks/`](.claude/hooks/).
- **The orientation file is the orientation file.** [`CLAUDE.md`](CLAUDE.md) loads on every session and tells Claude how this repo works. There is no second README that drifts from the first.

You can absolutely work in this repo without Claude — the specs, tests, and code are all human-readable, the docs site renders the spec library, and `mise` runs everything from the terminal. But the workflows assume Claude is doing a lot of the typing.

## Native everywhere: spec-as-framework, not lowest-common-denominator

The other half of this template's thesis: **each platform is built in its own native language, framework, and tooling — and the spec is what holds them together instead of a shared runtime.**

The conventional way to ship "the same app" across web, mobile, and desktop is to pick a cross-platform framework — React Native, Flutter, Capacitor, Kotlin Multiplatform — and accept the trade-offs that come with it: a thin abstraction over each platform's UI layer, generic interaction patterns, a JS/Dart/Kotlin runtime layered on top of the OS, and a shared codebase that is structurally biased toward whatever the framework finds easy to express. You ship faster initially, you ship more uniformly forever, and the app feels approximately right on every platform.

This template makes the opposite bet:

- **Web is React + TanStack Start + Convex + Tailwind v4 + React Aria.** Server functions, reactive Convex queries, the actual web platform — and the reference every other client mirrors.
- **Websites are Astro + React islands.** Content-first marketing and docs surfaces that ship HTML and hydrate sparingly.
- **Apple is UIKit + Observation + SwiftData.** One Swift codebase across iOS / iPadOS / macOS / tvOS / watchOS / visionOS — UIKit on iOS · iPadOS · tvOS · visionOS, AppKit on macOS, SwiftUI on watchOS; native navigation, gestures, accessibility, the HIG as a real constraint.
- **Android is Jetpack Compose + Material 3 + Kotlin coroutines/Flow + Room.** Material You theming, predictive back, the real Android system behaviors.
- **Windows is C# + WinUI 3 + MVVM Toolkit + EF Core.** Native XAML, the real Windows shell.
- **Linux is Rust + GTK 4 + Adwaita + Relm4.** Native GNOME, the real desktop.
- **The CLI is one of Node (TS-Rest + Bombshell), Rust (Clap + charmed_rust), or Go (Cobra/Fang + Charm).** Headless automation and/or a rich TUI.
- **Backend is Convex (Clerk for auth), reached contract-first.** Web and the website talk to Convex directly through its TypeScript client — reactive subscriptions, mutations, codegen-driven types. Native and CLI clients consume a **generated OpenAPI client** (Swift OpenAPI Generator, Kotlin OpenAPI Generator, Kiota, Progenitor, oapi-codegen) over the platform's own HTTP stack, with an on-device database (SwiftData / Room / EF Core / Diesel) as a local-first cache. No platform hand-rolls a transport or mirrors the protocol by hand — the contract is the only thing that crosses the wire.

There is no shared package between any of these. There is no transpilation step, no bridge layer, no abstracted UI primitive. Each app ships the platform's native idioms — the kind of detail that distinguishes "an app" from "a website wrapped in a chrome." When something is genuinely different between platforms (a swipe gesture, a system share sheet, a context menu, a haptic), the platform implements it the platform's way, marked `// SPEC: <id> (deviates: <reason>)` so that divergence is explicit rather than smuggled.

**One product, or several related ones.** Most of this document describes the common case: one product, projected across platforms, unified by specs. But a monorepo here can also hold _several logically-related apps_ — not just native projections of a single product, but distinct apps that share specs at some level (a common domain, shared cross-cutting conventions) and, where they run on the same stack, actual code. They're disambiguated by **name**, not platform alone: if you need a second CLI, that's a second named app with its own platform projections — not a second per-language CLI folder bolted onto the first. The no-shared-code rule is specifically about _cross-platform_ projections of one app; ordinary library reuse between same-runtime sibling apps is just good engineering.

**The spec is what fills the role a shared framework usually plays.** It is the contract that says "every client must support _these_ states, _these_ transitions, _these_ errors, _these_ acceptance criteria." Each platform satisfies the contract idiomatically. The spec is the framework — written in markdown, enforced by reverse pointers and Gherkin scenarios, kept honest by `/sdd-drift` and `/sdd-verify`.

Why is this tractable now when historically it wasn't? Because **agents close the cost gap**. Implementing a feature once per platform across a half-dozen native stacks used to be prohibitively expensive — N times the engineering effort, N sources of bugs, N drift trajectories. With Claude doing most of the per-platform translation from a shared spec, the cost flattens dramatically. You write the spec once, dispatch `/sdd-apply <spec-id> <platform>` for each target, and the agent produces idiomatic native code on each platform that satisfies the same behavioral contract. Drift is detected mechanically; reconciliation is a command, not a project.

The result: an app that feels at home on every platform — not a uniform skin over a uniform runtime — without paying the historical cost of writing and maintaining a fleet of native apps by hand.

## Why this template invents its own skills, agents, hooks, and conventions

There are several mature ecosystems for AI-assisted development — [**Superpowers**](https://github.com/obra/superpowers) (a curated skill library) and [**Beads**](https://github.com/steveyegge/beads) (an issue tracker designed for AI workflows) being two we drew inspiration from. We deliberately don't depend on either. Here's why.

### vs. Superpowers

Superpowers is a fantastic, broad skill library — debugging, brainstorming, code review, plan execution, worktree management, and more. We took the **patterns** but rewrote them lighter and tighter for this template's specific shape:

- **No plan documents, no branch ceremony.** Superpowers leans on plan files and worktree branches to coordinate multi-step work. This template uses TodoWrite + per-spec subagents (see the [`implementing-a-spec`](.claude/skills/implementing-a-spec/SKILL.md) skill) because the unit of work is "satisfy this spec on this platform" — finer-grained than a plan doc, larger than a single edit.
- **Reverse pointers replace task tracking.** Every line of code points back to its spec via `// SPEC: <id>`. Drift detection is `rg`-able, not ticket-shaped. You don't need a separate issue tracker to know what's done — `/sdd-drift` and `/sdd-cover` derive it from the code.
- **Smaller surface area = faster onboarding.** Superpowers ships ~50 skills. This template ships a focused set tuned to the spec-driven workflow — the cross-cutting process skills plus one development skill per platform. Run `/setup` and the platform skills you don't use are pruned, so each copy stays readable in a sitting.
- **`[NEEDS CLARIFICATION]` is the missing-info convention.** Borrowed in spirit from Superpowers' brainstorming flow but adapted: any unresolved question lives inline in the spec as `[NEEDS CLARIFICATION: ...]` and is resolved by the [`/sdd-clarify`](.claude/commands/sdd-clarify.md) command.

Several skills (notably [`brainstorming-feature`](.claude/skills/brainstorming-feature/SKILL.md), [`test-driven-development`](.claude/skills/test-driven-development/SKILL.md), [`systematic-debugging`](.claude/skills/systematic-debugging/SKILL.md), [`verification-before-completion`](.claude/skills/verification-before-completion/SKILL.md)) lift their core moves directly from Superpowers — see the attribution in each `SKILL.md` header. The originals are excellent; we just wanted them shaped to this repo's grain.

### vs. Beads

Beads is a thoughtful ticket tracker built around the way agents actually work — claim/release semantics, dependency graphs, ready-task surfacing. We chose to **derive the same information from the spec library itself** instead:

- **The spec ID _is_ the ticket ID.** When `vm.items.list` exists in `specs/`, that is the source of work. Tracking it separately in a ticket system means two systems to keep in sync — and the spec already has frontmatter (`id`, `kind`, `depends-on`, `[NEEDS CLARIFICATION]`) that subsumes most ticket fields.
- **"Ready work" is `/sdd-drift` + `/sdd-cover`.** Specs that exist but lack implementation, or whose implementation has drifted, are the ready queue. The [`drift-hunter`](.claude/agents/drift-hunter.md) subagent produces a prioritized punch list on demand.
- **Cross-platform coverage is structural, not a query.** Every spec maps to N platforms; `/sdd-cover <spec-id>` shows you which platforms implement it and which tests pass. No ticket joins required.
- **Fewer dependencies for a template.** Bringing in a tracker means everyone who clones the template installs and configures it before doing useful work. Specs and reverse pointers are just markdown and grep.

If your project _grows_ to need a ticket tracker (especially for cross-team coordination beyond the repo), Beads is a great choice — it composes cleanly alongside this template. We just didn't want it to be a precondition.

The one class of work that _isn't_ derivable from the spec library is **sub-spec defects**: platform-local cosmetic / polish / quirk issues that the cross-platform spec deliberately doesn't speak to. For those, each platform keeps an [`apps/<platform>/DEFECTS.md`](.claude/templates/platform/DEFECTS.md) drain file — filed via [`/sdd-defect`](.claude/commands/sdd-defect.md), drained via the [`triaging-defects`](.claude/skills/triaging-defects/SKILL.md) skill, deleted on fix. It's a tiny convention with no status fields, severity labels, or assignees on purpose: the file wants to be empty, and the fix commit is the durable record.

### vs. generic tooling

Most "AI-friendly" templates are AI-agnostic templates with a `CLAUDE.md` bolted on. This one inverts that: the spec format, the slash commands, the hook lifecycle, and the per-platform discipline were all designed assuming you'll be reading and writing markdown _alongside_ an agent that can navigate the repo. If we end up using a different agent later, much of the structure will still hold — but the optimization target is Claude Code today.

## How SDD works here

The flow is the same on every platform:

1. **Brainstorm a feature** — invoke the [`brainstorming-feature`](.claude/skills/brainstorming-feature/SKILL.md) skill. It walks narrative → stories → models → view-models → flows → errors and populates a `features/<NNNN>-<slug>/` folder using the templates in [`.claude/templates/feature/`](.claude/templates/feature/). Anything unresolved becomes `[NEEDS CLARIFICATION: ...]`.
2. **Clarify** — run [`/sdd-clarify <feature>`](.claude/commands/sdd-clarify.md) to resolve outstanding markers with you, the human.
3. **Review the spec** — dispatch the [`spec-reviewer`](.claude/agents/spec-reviewer.md) subagent for a P0/P1/P2 audit before implementation.
4. **Implement on the reference platform** — invoke the [`implementing-a-spec`](.claude/skills/implementing-a-spec/SKILL.md) skill, or run [`/sdd-apply <spec-id> web`](.claude/commands/sdd-apply.md). The skill writes failing tests first ([`test-driven-development`](.claude/skills/test-driven-development/SKILL.md)), then the minimum implementation to pass them, then runs spec-compliance and code-quality reviews.
5. **Mirror to other platforms** — `/sdd-apply <spec-id> ios`, `/sdd-apply <spec-id> android`. The web implementation becomes a worked example alongside the spec; the agent translates idiomatically.
6. **Verify** — `/sdd-verify <platform>` runs the platform's behavioral tests. The [`visual-verifier`](.claude/agents/visual-verifier.md) subagent walks the Gherkin scenarios through the actual UI (Chrome DevTools / iOS simulator / Android emulator).
7. **Audit drift over time** — `/sdd-drift <platform>` (or the [`drift-hunter`](.claude/agents/drift-hunter.md) subagent for a multi-platform sweep) lists spec IDs whose implementation has drifted.

The discipline is enforced by hooks: `block-generated.sh` refuses edits to generated artifacts, `format-on-edit.sh` formats every touched file, `spec-reconcile.sh` reminds you to `/sdd-apply` when a spec changes, and `stop-lint.sh` runs lint on dirty platforms before letting Claude declare done.

### The standard flow, command by command

The prose above is the shape; this is the runbook. It's the sequence to run **per feature**, with concrete arguments. The example feature is `0001-managing-items`, whose folder contains the specs `domain.item`, `vm.items.list`, and `story.item.create`.

**Phase 0 — once, on a fresh copy of the template**

```text
/setup                              # choose platforms + backend; prune the rest
```

**Phase 1 — author the feature (specs only; no code yet)**

```text
invoke skill: brainstorming-feature # narrative → stories → models → view-models → flows → errors
                                    #   ↳ writes features/0001-managing-items/
/sdd-clarify 0001                   # resolve every [NEEDS CLARIFICATION] marker with you
/sdd-analyze 0001                   # read-only: gaps, contradictions, dangling depends-on
dispatch subagent: spec-reviewer    # P0/P1/P2 audit of the spec before any code
```

Loop Phase 1 until `/sdd-analyze` is clean and the spec-reviewer has no P0/P1 issues. **The spec is the leverage — sharpen it here, before it forks across N platforms.**

**Phase 2 — implement on the reference platform (web)**

Apply each spec in the feature in `depends-on` order — models, then view-models, then stories/errors:

```text
/sdd-apply domain.item       web    # writes failing tests first, then minimum impl
/sdd-apply vm.items.list     web
/sdd-apply story.item.create web
/sdd-verify web                      # run web's behavioral suite, keyed by spec ID
dispatch subagent: visual-verifier   # walk the Gherkin scenarios through the real UI
```

`/sdd-apply` runs the [`implementing-a-spec`](.claude/skills/implementing-a-spec/SKILL.md) loop internally (test → impl → spec-compliance review → code-quality review → adversarial pass) and commits at natural boundaries. If it reports the spec itself is wrong, stop and fix the spec, then re-apply.

**Phase 3 — mirror to every other platform**

Same spec IDs, same order, once per target platform. Web is now a worked example alongside the spec:

```text
/sdd-apply domain.item       ios
/sdd-apply vm.items.list     ios
/sdd-apply story.item.create ios
/sdd-verify ios

/sdd-apply domain.item       android
/sdd-apply vm.items.list     android
/sdd-apply story.item.create android
/sdd-verify android
#   … repeat per platform: website · windows · linux · cli
```

**Phase 4 — confirm coverage, then keep it honest over time**

```text
/sdd-cover story.item.create        # which platforms implement it + which tests pass
/sdd-drift web                       # periodically, per platform: what's gone stale
/sdd-reconcile ios                   # when a platform raced ahead — fold its change back into the spec + others
/sdd-defect ios "list scrollbar flickers on first paint"   # capture platform-local polish; drain via triaging-defects
```

**Two recurring sub-flows once the feature exists:**

- **Changing a spec** → edit the spec file, then `/sdd-apply <id> <platform>` for every platform that implements it. The `spec-reconcile.sh` hook reminds you which ones.
- **Fixing a platform-only bug the spec already requires** → just fix it and `/sdd-verify <platform>`. No spec edit. (If the _behavior_ changed rather than getting _corrected_, that's `/sdd-reconcile` instead.)

You drive the commands; the hooks (format-on-edit, codegen, lint-on-stop) and the per-spec review stages run on their own. The throughput is in Phases 2–3 fanning out across platforms; the judgment is in Phase 1.

## Repo layout

```
.
├── CLAUDE.md                          ← orientation doc Claude loads every session
├── README.md                          ← this file (orientation for humans)
├── .env.schema                        ← Varlock environment contract
├── .mcp.json                          ← project-level MCP servers (Chromium DevTools)
├── .claude/                           ← everything Claude-shaped (see catalog below)
├── docs/                              ← VitePress site rendering specs/ and features/
├── specs/                             ← cross-cutting specs (CONVENTIONS, ARCHITECTURE, DESIGN_SYSTEM, STACK)
├── mise.toml                          ← root task runner (docs:* + per-platform orchestration)
│
├── features/                          ← (you create) feature-scoped specs as <NNNN>-<slug>/
├── apps/                              ← (you create) platform implementations
│   ├── web/                           ←   React + TanStack Start + Convex (reference)
│   ├── website/                       ←   Astro + React islands
│   ├── ios/                           ←   Swift / UIKit / SwiftData (Apple family)
│   ├── android/                       ←   Kotlin / Jetpack Compose / Room
│   ├── windows/                       ←   C# / WinUI 3 / EF Core
│   ├── linux/                         ←   Rust / GTK 4 + Adwaita / Relm4
│   └── cli/                           ←   the CLI — one stack: Node (TS-Rest) · Rust (charmed_rust) · Go (Charm)
└── services/                          ← (you create) backend services
    └── convex/                        ←   Convex backend + Clerk auth
```

The `apps/` and `services/` directories aren't committed in this template — you scaffold them per-platform as you start that platform's work. Each scaffolded directory will have its own `CLAUDE.md` (with stack idioms) and `mise.toml` (with build/test/lint tasks).

## Catalog of Claude-specific files

Everything below is what makes this template "Claude-native." If you copied just these files into another repo, you'd have most of the spec-driven workflow.

### Root-level

| Path                         | Purpose                                                                                                                                                                                                                                                                                                                                      |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`CLAUDE.md`](CLAUDE.md)     | Loaded on every Claude Code session. The top-level orientation: how the repo works, where to read first, slash command index, skill index. `@includes` the rule files below so they're part of every session.                                                                                                                                |
| [`STACK.md`](specs/STACK.md) | The canonical toolchain catalog — every tool, framework, and service this template knows how to wire up, by layer. The superset `/setup` prunes from.                                                                                                                                                                                        |
| [`.env.schema`](.env.schema) | [Varlock](https://varlock.dev) environment contract: the committed, typed declaration of the project's env vars. Real values live in gitignored `.env` / `.env.local`.                                                                                                                                                                       |
| [`.mcp.json`](.mcp.json)     | Project-level MCP server config. Registers the [Chrome DevTools MCP](https://github.com/ChromeDevTools/chrome-devtools-mcp) pointed at **Chromium** (not Chrome) in `--isolated` mode for web visual verification. Per-platform IDE bridges (Xcode, Android Studio/JetBrains, Roslyn) are configured in **user/local** MCP config, not here. |
| `apps/<platform>/CLAUDE.md`  | Per-platform orientation (created when you scaffold the platform). Stack idioms, test commands, where reverse pointers go in that language.                                                                                                                                                                                                  |
| `services/convex/CLAUDE.md`  | Backend orientation (created when you scaffold Convex). Schema-as-protocol conventions, mutation/query patterns.                                                                                                                                                                                                                             |

### `.claude/settings.json`

Project-level Claude Code settings. Wires up:

- **Permissions** — auto-allow safe read-only and build commands (`mise`, `pnpm`, `cargo`, `dotnet`, `xcodebuild`, `adb`, `rustfmt`, `oxfmt`, `rg`, etc.); ask before `git push`, `convex deploy`, `wrangler deploy`, `railway up`, `rm -rf`.
- **Hooks** — registers every script in `.claude/hooks/` to its lifecycle event (`PreToolUse`, `PostToolUse`, `Stop`, `UserPromptSubmit`, `Notification`).
- **Default permission mode** — `auto` (continuous execution).

### `.claude/rules/` — `@included` into orientation docs

Loaded on every session via `@includes` from `CLAUDE.md`.

| Path                                                                       | Purpose                                                                                                        |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [`.claude/rules/code-quality.md`](.claude/rules/code-quality.md)           | The "what good code looks like in this repo" rules — file size, naming, comments, abstraction, error handling. |
| [`.claude/rules/commit-discipline.md`](.claude/rules/commit-discipline.md) | When to commit, what one commit contains, message format, staging policy, push policy.                         |
| [`.claude/rules/spec-conventions.md`](.claude/rules/spec-conventions.md)   | The compact between specs, tests, and implementations. Loaded by every per-platform `CLAUDE.md`.               |

### `.claude/skills/` — procedural workflows

Skills are markdown files that encode "how we do X here." Claude invokes them via the `Skill` tool.

| Skill                                                                                      | When to use                                                                                                                                                    |
| ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`brainstorming-feature`](.claude/skills/brainstorming-feature/SKILL.md)                   | Before starting any new feature. Walks narrative → stories → models → view-models → flows → errors.                                                            |
| [`writing-user-stories`](.claude/skills/writing-user-stories/SKILL.md)                     | When authoring or reviewing a story file. Enforces Gherkin discipline.                                                                                         |
| [`implementing-a-spec`](.claude/skills/implementing-a-spec/SKILL.md)                       | The default "how to write code" workflow. Per-spec subagent dispatch + two-stage review. Used by `/sdd-apply`.                                                 |
| [`test-driven-development`](.claude/skills/test-driven-development/SKILL.md)               | When writing any production code. Iron Law: no production code without a failing test first.                                                                   |
| [`verification-before-completion`](.claude/skills/verification-before-completion/SKILL.md) | Before claiming any work is complete. Run the verifying command in this turn; evidence before claims.                                                          |
| [`systematic-debugging`](.claude/skills/systematic-debugging/SKILL.md)                     | When encountering any bug or unexpected behavior. Find the root cause before proposing a fix.                                                                  |
| [`triaging-defects`](.claude/skills/triaging-defects/SKILL.md)                             | When `apps/<platform>/DEFECTS.md` is non-empty. Classify each entry as fix-in-place / promote-to-spec / won't-fix and drain.                                   |
| [`web-development`](.claude/skills/web-development/SKILL.md)                               | When writing web-app code. React + TanStack suite + Convex + Clerk + Tailwind + React Aria + Motion + Zod idioms.                                              |
| [`web-verification`](.claude/skills/web-verification/SKILL.md)                             | When verifying web UI in a browser. Wraps the Chrome DevTools MCP.                                                                                             |
| [`website-development`](.claude/skills/website-development/SKILL.md)                       | When writing the marketing/content site. Astro + React islands + content collections idioms.                                                                   |
| [`ios-development`](.claude/skills/ios-development/SKILL.md)                               | When writing Apple-family code. UIKit (AppKit on macOS, SwiftUI on watchOS) + Observation + SwiftData + Swift Testing + generated OpenAPI client.              |
| [`ios-simulator-control`](.claude/skills/ios-simulator-control/SKILL.md)                   | When verifying Apple UI changes. Wraps `xcrun simctl` + `idb`.                                                                                                 |
| [`android-development`](.claude/skills/android-development/SKILL.md)                       | When writing Android code. Compose + Material 3 + coroutines/Flow + Room + Ktor (OkHttp) + OpenAPI idioms.                                                     |
| [`android-emulator-control`](.claude/skills/android-emulator-control/SKILL.md)             | When verifying Android UI changes. Wraps `adb` + `uiautomator`.                                                                                                |
| [`windows-development`](.claude/skills/windows-development/SKILL.md)                       | When writing Windows code. C# + WinUI 3 + MVVM Toolkit + EF Core + Kiota; Fluent Design; Roslyn MCP bridge.                                                    |
| [`windows-app-control`](.claude/skills/windows-app-control/SKILL.md)                       | When verifying Windows UI changes **on Windows**. Wraps the `winapp` CLI (`winapp run` + `winapp ui`). Inert on a macOS-hosted agent.                          |
| [`linux-development`](.claude/skills/linux-development/SKILL.md)                           | When writing Linux desktop code. Rust + GTK 4 + Adwaita + Relm4 + Diesel + Progenitor idioms.                                                                  |
| [`node-cli-development`](.claude/skills/node-cli-development/SKILL.md)                     | When writing the **Node** CLI stack (`apps/cli`). TS-Rest + Bombshell + Drizzle + plainjob; hosts the OpenAPI contract in OpenAPI mode.                        |
| [`rust-cli-development`](.claude/skills/rust-cli-development/SKILL.md)                     | When writing the **Rust** CLI stack (`apps/cli`). Clap + charmed_rust (bubbletea/bubbles/lipgloss/huh/glamour/harmonica/wish) + Diesel + reqwest + Progenitor. |
| [`go-cli-development`](.claude/skills/go-cli-development/SKILL.md)                         | When writing the **Go** CLI stack (`apps/cli`). Cobra/Fang + Bubble Tea + Bubbles + Lip Gloss + Huh + Glamour + database/sql (go-sqlite) + oapi-codegen.       |

### `.claude/agents/` — cross-cutting subagents

Subagents run in their own context window and return a single message back to the main conversation. Use them for audits and isolated checks.

| Agent                                                  | Purpose                                                                                                                                                             |
| ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`drift-hunter`](.claude/agents/drift-hunter.md)       | Audits cross-platform spec/impl drift. Runs `/sdd-drift` across platforms, cross-references with `/sdd-verify` output, returns a prioritized punch list. Read-only. |
| [`spec-reviewer`](.claude/agents/spec-reviewer.md)     | Reviews a spec file before it lands. Frontmatter, Gherkin discipline, `[NEEDS CLARIFICATION]` markers, reverse-pointer health. P0/P1/P2 issue list.                 |
| [`test-gap-finder`](.claude/agents/test-gap-finder.md) | Finds Gherkin scenarios that don't have a matching `[scenario.<id>]`-tagged test on a given platform. Test-coverage drift (vs. drift-hunter's code drift).          |
| [`visual-verifier`](.claude/agents/visual-verifier.md) | Drives Chrome DevTools / iOS simulator / Android emulator through each Gherkin scenario in a `story.*` spec, screenshots each state, reports rendering mismatches.  |
| [`handoff-builder`](.claude/agents/handoff-builder.md) | At the end of a development pass, generates or updates `HANDOFF.md` so a future session can pick up the branch with full context.                                   |

### `.claude/commands/` — slash commands

User-typed commands. Each is intent-only at the moment — the agent uses `rg`, `Edit`, `AskUserQuestion`, etc. to fulfill them; no automation script behind them yet.

| Command                                                                 | Purpose                                                                                           |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| [`/setup`](.claude/commands/setup.md)                                   | **Run once on a fresh copy.** Choose which platforms + backend you ship; prune everything else.   |
| [`/sdd-apply <spec-id> <platform>`](.claude/commands/sdd-apply.md)      | Regenerate a spec's implementation and tests on a target platform.                                |
| [`/sdd-verify <platform>`](.claude/commands/sdd-verify.md)              | Run the platform's behavioral test suite and report which spec IDs pass.                          |
| [`/sdd-drift <platform>`](.claude/commands/sdd-drift.md)                | List spec IDs whose implementation has drifted from the spec on a platform.                       |
| [`/sdd-cover <spec-id>`](.claude/commands/sdd-cover.md)                 | Show which platforms implement a spec and which of their tests pass.                              |
| [`/sdd-reconcile <source-platform>`](.claude/commands/sdd-reconcile.md) | Bring the spec + other platforms in line with this platform's impl (when a platform raced ahead). |
| [`/sdd-clarify <feature-or-spec>`](.claude/commands/sdd-clarify.md)     | Scan a feature or spec for `[NEEDS CLARIFICATION]` markers and resolve them with the user.        |
| [`/sdd-analyze <feature>`](.claude/commands/sdd-analyze.md)             | Read-only cross-artifact consistency check for a feature folder.                                  |
| [`/sdd-defect <platform> <desc>`](.claude/commands/sdd-defect.md)       | File a sub-spec defect into `apps/<platform>/DEFECTS.md` without breaking flow.                   |

### `.claude/hooks/` — lifecycle scripts

Bash scripts wired up in `settings.json`. Each runs at a specific Claude Code lifecycle event. Failures are logged but don't crash the agent (except where blocking is intentional, e.g. `stop-lint.sh`).

| Hook                                                             | Event                                       | Purpose                                                                                                                                                                                                                                                                                                         |
| ---------------------------------------------------------------- | ------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`block-generated.sh`](.claude/hooks/block-generated.sh)         | `PreToolUse` (Edit/Write/MultiEdit)         | Refuses edits to tool-generated files (Convex `_generated/`, Xcode-derived data, Cargo `target/`, .NET `obj/`+`bin/`, Astro `.astro/`, etc.).                                                                                                                                                                   |
| [`scoped-commits.sh`](.claude/hooks/scoped-commits.sh)           | `PreToolUse` (Bash, gated to `git commit*`) | Requires a [Scoped Commits](https://scopedcommits.com/) subject (`<scope>: <description>`) whose scope is **defined** — a spec/feature `id:` (a reverse pointer), an `apps/*`/`services/*` dir, a harness area, a name in `.claude/commit-scopes`, or `treewide`. Rejects the Conventional `type(scope):` form. |
| [`format-on-edit.sh`](.claude/hooks/format-on-edit.sh)           | `PostToolUse` (Edit/Write/MultiEdit)        | Formats the touched file by dispatching to its platform's `fmt` mise task (`mise run -C <dir> fmt -- <file>`). The formatter lives in each platform's `fmt` task, not the hook, so adding a platform never touches it.                                                                                          |
| [`convex-codegen.sh`](.claude/hooks/convex-codegen.sh)           | `PostToolUse` (Edit/Write/MultiEdit)        | Regenerates Convex types when `schema.ts` changes.                                                                                                                                                                                                                                                              |
| [`openapi-codegen.sh`](.claude/hooks/openapi-codegen.sh)         | `PostToolUse` (Edit/Write/MultiEdit)        | When the OpenAPI contract changes, reminds you to regenerate the per-platform typed clients (reminder only — the producer is a project choice).                                                                                                                                                                 |
| [`tuist-regen.sh`](.claude/hooks/tuist-regen.sh)                 | `PostToolUse` (Edit/Write/MultiEdit)        | Regenerates the Xcode project when `Project.swift` changes.                                                                                                                                                                                                                                                     |
| [`spec-reconcile.sh`](.claude/hooks/spec-reconcile.sh)           | `PostToolUse` (Edit/Write/MultiEdit)        | When a spec is edited, lists implementations that reference its ID and suggests `/sdd-apply` per platform. When code is edited, surfaces drift hints.                                                                                                                                                           |
| [`stop-lint.sh`](.claude/hooks/stop-lint.sh)                     | `Stop`                                      | Runs lint on whichever platforms have uncommitted changes since `HEAD`. **Blocks the stop** if any lint fails — Claude can't declare "done" with a dirty lint.                                                                                                                                                  |
| [`user-prompt-context.sh`](.claude/hooks/user-prompt-context.sh) | `UserPromptSubmit`                          | Injects current branch + uncommitted changes into the agent's context, so natural commit points are obvious.                                                                                                                                                                                                    |
| [`notify-long-task.sh`](.claude/hooks/notify-long-task.sh)       | `Notification`                              | Surfaces a macOS notification when Claude Code needs attention (long-running task, permission prompt).                                                                                                                                                                                                          |

### `.claude/templates/` — canonical scaffolds

Markdown templates for new specs and features. Used by the `brainstorming-feature` skill and by anyone authoring a spec by hand.

| Path                                                                             | Purpose                                                                                                                                                                                                              |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`.claude/templates/feature/`](.claude/templates/feature/)                       | Full structure for a new `features/<NNNN>-<slug>/` folder: `NARRATIVE.md`, `stories/STORY.md`, `models/MODEL.md`, `view-models/VIEW_MODEL.md`, `use-cases/USE_CASE.md`, `user-flow/USER_FLOW.md`, `errors/ERROR.md`. |
| [`.claude/templates/spec/MODEL.md`](.claude/templates/spec/MODEL.md)             | Canonical template for a cross-cutting spec under `specs/`.                                                                                                                                                          |
| [`.claude/templates/platform/DEFECTS.md`](.claude/templates/platform/DEFECTS.md) | Seed file for per-platform sub-spec defect tracking. Copied to `apps/<platform>/DEFECTS.md` on first `/sdd-defect`.                                                                                                  |

## Local tooling

[`mise`](https://mise.jdx.dev) manages toolchains and runs tasks. The root [`mise.toml`](mise.toml) ships `docs:*` tasks and `fmt`; per-platform tasks live in `apps/*/mise.toml` and `services/*/mise.toml` once you scaffold them. Environment variables are managed by [Varlock](https://varlock.dev) against the committed [`.env.schema`](.env.schema) — run env-dependent commands with `varlock run -- <cmd>`.

```sh
mise run docs:dev          # docs site (VitePress) at http://localhost:5173
mise run docs:build        # static build to docs/.vitepress/dist
mise run docs:preview      # preview the built site
mise run fmt               # format the entire project (oxfmt)
mise tasks                 # list everything available
```

When you scaffold a platform, define `fmt` and `lint` tasks in its `mise.toml` (its formatter / linter — `fmt` accepts optional file paths) so the `format-on-edit` and `stop-lint` hooks can call them, and add its orchestration task at the root (e.g. `web:dev`, `ios:test`) so cross-platform commands work from the repo root.

## What's deliberately not included

- **`apps/` and `services/` directories.** Scaffold these when you choose your stack — different teams will want different platforms in different orders.
- **Automation behind the `/sdd-*` commands.** They are intent-only at the moment; the agent uses `rg`, `Edit`, `AskUserQuestion`, etc. to fulfill them. As patterns stabilize, some of this will move into shell scripts or a small CLI.
- **A worked example feature.** The original repo this was extracted from has a contacts-app pass; this template ships clean so you can put your own thing in `features/0001-*/`.
- **A ticket tracker.** As discussed above — the spec library is the source of work. Bring in Beads or your tracker of choice if you outgrow that.

## Read next

- [`CLAUDE.md`](CLAUDE.md) — the orientation doc Claude Code loads on every session. Read this even if you're working without an agent; it's the canonical "how this repo works."
- [`specs/CONVENTIONS.md`](specs/CONVENTIONS.md) — the spec contract: IDs, kinds, frontmatter, reverse pointers, deviation markers, drift detection.
- [`specs/ARCHITECTURE.md`](specs/ARCHITECTURE.md) — top-level layering, data flow, deployment.
- [`specs/DESIGN_SYSTEM.md`](specs/DESIGN_SYSTEM.md) — design tokens, component vocabulary, parity rules across platforms.
- [`.claude/skills/brainstorming-feature/SKILL.md`](.claude/skills/brainstorming-feature/SKILL.md) — how to author a feature folder.
- [`.claude/skills/implementing-a-spec/SKILL.md`](.claude/skills/implementing-a-spec/SKILL.md) — the default "how to write code" workflow.
