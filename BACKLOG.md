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

**Platform dev skills (9):** android · go-cli · ios · linux · node-cli · rust-cli · web · website · windows -development. 🔒 Blocked on the targets/`specs.jsonc` config + a platform-pack projection decision.

**Platform verification/control skills (4):** android-emulator-control · ios-simulator-control · web-verification · windows-app-control. 🔒 Same dependency.

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
