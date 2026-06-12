# SpecKit backlog

Running list of follow-ups raised while building the fork, so nothing gets lost
between sessions. Newest asks at the top of each section. Status: ✅ done ·
🔄 in progress · ⬜ todo · 🔒 blocked (dependency noted).

---

## Done this session

- ✅ **Command-prompt synthesis** — all 9 `/speckit.*` commands reworked: Workbench's hard-won
  discipline (via the skills they invoke) + spec-kit's structural rigor (prioritized P1/P2/P3
  stories, measurable success criteria, the clarify/analyze taxonomies, checklists, constitution)
  on the fork's tooling (`.speckit/`, the `specify` engine, no scripts). All stale upstream refs
  removed. Folder-layout decision resolved (see "Command-prompt rework" below).
- ✅ **Authoring skills ported** — `brainstorming-feature`, `writing-user-stories`,
  `implementing-a-spec`; the commands invoke them.
- ✅ **Process-discipline trio** — `test-driven-development`, `verification-before-completion`,
  `adversarial-review` authored and projected by `init` into the agent's skills dir.
- ✅ **`systematic-debugging`** skill ported (command-agnostic).
- ✅ **Homebrew via native bottling** — dropped goreleaser's `brews:` download-formula;
  from-source `specify.rb` + tap auto-bump (`update-specify.yml`) + cross-repo dispatch
  from `release.yml` via the PAT. Artifacts in `packaging/homebrew/`. Deploy gated on first release.
- ✅ **Copilot skills → `.github/skills`** (cloud-agent convention).
- ✅ **`verify` config `command` is a string**, not an array.

---

## Skills port — bring over Workbench's full set

Workbench has **21 skills**; the slash commands should invoke them (as `/sdd-*` does).

**Universal process skills (8):**
- ✅ test-driven-development · verification-before-completion · adversarial-review · systematic-debugging
- ✅ implementing-a-spec · brainstorming-feature · writing-user-stories (ported + wired to the commands)
- 🔒 **triaging-defects** — the `DEFECTS.md` drain; blocked on establishing a defect-ledger convention + a `/speckit.defect` equivalent + the per-target folder model.

**Platform dev (9) + verification/control (4) skills:** ✅ All 13 ported to
`internal/coreassets/templates/packs/<stack>/` and projected **on demand** by `specify packs`,
gated on each target's `stack` (web/apple/android/windows/linux/go-cli/node-cli/rust-cli/website).
`init` stays process-skills-only. See [docs/config.md](docs/config.md#platform-packs).

**Wire skills to slash commands** ✅ — `/speckit.specify` → brainstorming-feature (+ writing-user-stories); `/speckit.implement` → implementing-a-spec; `/speckit.analyze` → `specify scan` + semantic passes; etc.

⬜ **Feature-folder templates** (minor) — the fork ships spec-kit's `spec-template`/`plan-template`/`tasks-template`, but the commands now author Workbench-style feature folders. The skills point at `specs/CONVENTIONS.md` for structure (works), but `NARRATIVE`/story/model/view-model/error templates under `.speckit/templates/feature/` would scaffold faster.

---

## Command-prompt rework ✅ (done this session)

All 9 `/speckit.*` commands (analyze · checklist · clarify · constitution · implement · plan ·
specify · tasks · taskstoissues) reworked to the fork's reality — synthesizing Workbench's
discipline with spec-kit's rigor:

- `.speckit/` not `.specify/`; no shell scripts; the `specify` engine (scan/verify/drift/cover/parity/gate).
- Structured args (`/speckit.plan 0001-feature-name web`); commands invoke the process skills.
- spec-kit strengths folded in: prioritized P1/P2/P3 stories, measurable success criteria, the
  clarify 5-question / analyze severity taxonomies, the "unit tests for requirements" checklist, the constitution.

**Folder-layout decision: resolved.** The Workbench data model (`features/<NNNN>/` with ID'd
`stories/`/`models/`/`view-models/`, `// SPEC:` pointers, scenario sub-IDs) is canonical per
`specs/CONVENTIONS.md` (mechanized in `specmodel`). spec-kit's monolithic `spec.md` is replaced by
the feature folder; `plan` and `tasks` become **per-platform layers on top** —
`features/<NNNN>/plans/<platform>.md` and `tasks/<platform>.md`.

---

## Stack scaffolding (new — proposed, needs direction)

⬜ `specify` should **scaffold a target's stack** — the wired-up starter (deps, pinned versions, config,
and the scenario-binding test harness) — so the agent doesn't reassemble the right packages/conventions
every time a new target or product begins. Complements the platform packs: a **pack** is the agent's
*guidance* for a stack; a **scaffold** is the runnable *starter* for it, on the SpecKit-recommended stack.

**Design doc:** [docs/design/stack-scaffolding.md](docs/design/stack-scaffolding.md) — full proposal, awaiting review.

**Decided:** SpecKit-**curated** templates (not delegated to `create-*`), rendered with Go `text/template`;
**design-first** (no code until the doc is signed off). Command: `specify target add <name> --stack <stack>`.
First slice = **web** end-to-end (green on `specify verify` immediately), then **apple** (exercises the
`swift` format + the `SpecTraits.swift` harness). Prior art: `~/Developer/Libraries/create-sprinkles`.

**Resolved:** plain `specs.json` (so the merge is a trivial load→add→write), keep `{{ }}` with escaping,
and `target add` runs the install (`--no-install` to skip). Folded into the design doc.

## Coverage gap — libraries / Swift packages / CLIs / extensions (from the ~/Developer sweep)

⬜ A sweep of `~/Developer` showed SpecKit is **app-centric** but a large slice of the real work isn't apps:
**libraries** (Reactivity, downpour, icing-components, content-layer, cider, sqlite-data, PrivateHeaderKit,
catalyst-remix, create-sprinkles), **Swift packages/CLIs** (remctl, apple-platform-tools — a monorepo of
Swift packages + CLIs), and a **VS Code extension** (mise-vscode). These don't fit the app-centric authoring
model (NARRATIVE → human user stories → view-models/flows) — a library's consumer is a developer.

**The engine doesn't care** (it joins scenario↔test regardless), so the fix is **additive**, not a redesign:
- A product **`kind: app | library`**. For `library`, relax `writing-user-stories`' "actor is human" → the
  actor is the API/CLI consumer; `story`+`domain`+`error` kinds apply, `view-model`/`flow` (UI) don't; dovetails
  with the property-test guidance already in TDD. Maybe an `api`/`contract` spec kind.
- New stacks/scaffolds: **`swift-package`**, **`swift-cli`** (SwiftPM, `swift test`, no simulator — distinct
  from the GUI `apple` stack), **`ts-lib`** (npm package: tsup/vite-lib + Vitest, no dev server), **`vscode-extension`**
  (vsce), maybe `browser-extension`. The binding harness (Swift Testing / Vitest) carries over unchanged.
- Monorepos (apple-platform-tools) are already covered — each package is a target/product.

**Decided: expand now.** Design folded into [docs/design/library-products.md](docs/design/library-products.md)
(the `kind` + the library authoring variant) and the [scaffolding doc](docs/design/stack-scaffolding.md)
(the `swift-package`/`swift-cli`/`ts-lib`/`vscode-extension`/`browser-extension` stacks). First build slice
stays **web**; library scaffolds + the authoring branch follow. No engine change.

## Config system — `.speckit/specs.json`

- ✅ **targets** — `.speckit/specs.json` (plain JSON) with version/agent/paths/targets;
  each target's verify wiring inline, retiring `.speckit/verify/<platform>.json`. The engine keys on
  **target** (lock `.speckit/lock/<target>/`); `scan` validates the config. Products are an optional
  label; the `products` collection and `contracts` are documented as futures in [docs/config.md](docs/config.md).
- ⬜ **product-rollup render** — `cover`/`parity` grouping + per-product verdict. The label and
  `ProductTargets()` exist; the render lands with the multi-target slice below.

---

## Subagents — claude-pack

- ✅ `spec-reviewer` · `test-gap-finder` · `drift-hunter` · `handoff-builder` ported and projected into
  `.claude/agents/` (claude-only — codex/generic/copilot have no projectable subagent-dispatch dir). Each
  leans on the engine: spec-reviewer → `specify scan`; the rest → `specify verify`/`drift`/`parity`.
- ✅ `visual-verifier` — ported (drives Chrome DevTools / iOS sim / Android emulator through a story's
  Gherkin scenarios), projected by `init` alongside the other subagents.
- ⬜ codex/copilot review-equivalents — open question; their delegation models differ from Claude's dispatch.

---

## Hooks (11) — claude-pack overlay

⬜ `block-generated` · `format-on-edit` · `scoped-commits` · `spec-reconcile` · `stop-lint` ·
`notify-long-task` · `user-prompt-context` + codegen hooks (`convex`/`openapi`/`tuist`). Note: `gate`
already mechanizes the enforcement ones (firewall / generated / scope) — decide which hooks remain
worthwhile vs. folded into `gate`.

---

## Scanner enhancement

⬜ Support the per-framework binding forms `CONVENTIONS.md` already documents: MSTest
`[TestProperty("scenario", …)]`, kotlin `@Tag("scenario:…")`, generic `// [scenario.id]` comment.
Currently the scanner reads only Swift traits + Vitest titles.

---

## Docs

⬜ Rewrite `spec-driven.md` — it's the stale upstream essay; make it reflect SpecKit (the fork's
data model, the engine, the discipline).

---

## Release gate (outward — needs explicit go-ahead)

⬜ The first release (`v0.1.0` tag) is the trigger that activates brew + mise. On tag: goreleaser
publishes archives and dispatches `specify-release` to the tap; the tap bumps + bottles. First-release
checklist is in `packaging/homebrew/README.md`. Do **not** tag or push to the tap without confirmation.
