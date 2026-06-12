# SpecKit backlog

Running list of follow-ups raised while building the fork, so nothing gets lost
between sessions. Newest asks at the top of each section. Status: ✅ done ·
🔄 in progress · ⬜ todo · 🔒 blocked (dependency noted).

---

## Done this session

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
- 🔒 **implementing-a-spec** — the orchestrator (impl → spec-compliance → code-quality → adversarial). Maps `/sdd-apply <id> <platform>` → `/speckit.implement <id> <platform>`. Blocked on the command rework below.
- 🔒 **brainstorming-feature** — authoring; blocked on the authoring folder-layout decision below.
- 🔒 **writing-user-stories** — story/Gherkin format; blocked on the folder-layout decision.
- 🔒 **triaging-defects** — the `DEFECTS.md` drain; blocked on establishing a defect-ledger convention + a `/sdd-defect` equivalent + the per-target folder model.

**Platform dev skills (9):** android · go-cli · ios · linux · node-cli · rust-cli · web · website · windows -development. 🔒 Blocked on the targets/`specs.jsonc` config + a platform-pack projection decision.

**Platform verification/control skills (4):** android-emulator-control · ios-simulator-control · web-verification · windows-app-control. 🔒 Same dependency.

**Wire skills to slash commands** 🔒 — `/speckit.specify` → brainstorming-feature + writing-user-stories; `/speckit.implement` → implementing-a-spec; etc. Part of the command rework.

---

## Command-prompt rework (BIG — newly found)

🔒 The `/speckit.*` command templates are **still upstream spec-kit verbatim**. `implement.md`
references `scripts/bash/check-prerequisites.sh`, `.specify/extensions.yml`, and `tasks.md` —
none exist in the fork. All 9 commands (analyze · checklist · clarify · constitution · implement ·
plan · specify · tasks · taskstoissues) need reworking to the fork's reality:

- `.speckit/` not `.specify/`; no shell scripts; the `specify` engine (scan/verify/drift/cover/parity/gate).
- The Workbench data model, not upstream's `tasks.md`.
- Structured args, e.g. `/speckit.plan 0001-feature-name web`.
- Invoke the process skills (above).

⬜ **Decide the authoring folder layout** (architecture call): upstream's `specs/<NNN>-feature/{spec,plan,tasks}.md`
vs Workbench's `features/<NNNN>/{NARRATIVE,stories/,models/,view-models/,user-flow/,errors/}`. Currently
contradictory — upstream templates (`spec-template`/`plan-template`/`tasks-template`) coexist with a
Workbench-style `specs/models/` library. This unblocks the three authoring skills.

---

## Config system — `.speckit/specs.jsonc`

⬜ products / contracts / targets / agent. Targets generate the verify adapters; the engine keys on
**"target"** rather than "platform". (`verify.command` as a string is already done.)

---

## Subagents (5) — claude-pack

⬜ `spec-reviewer` · `test-gap-finder` · `drift-hunter` · `handoff-builder` · `visual-verifier` →
project into `.claude/agents/` (+ equivalents). They mechanize the review stages that `implementing-a-spec`
and `adversarial-review` describe.

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
