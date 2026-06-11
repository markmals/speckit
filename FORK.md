# FORK.md

SpecKit is a hard fork of [github/spec-kit](https://github.com/github/spec-kit) (via `markmals/speckit`). It rewrites the executable CLI in Go, adopts the Workbench spec data model as core law (dotted stable IDs, kind taxonomy, scenario sub-IDs, reverse pointers, deviation markers), and adds the Mocha runtime engine (`scan`/`verify`/`drift`/`cover`/`parity`/`gate`/`lock`/`ledger`). **No upstream contribution is intended; divergence is the point.**

Never rebase on upstream. Cherry-pick _prompt-level_ improvements (command markdown, template language) opportunistically — those are data, cheap to take — and nothing else. The authoritative design document is `speckit-fork-plan.md` (decisions D1–D13, phases, executable exit criteria).

## Provenance

| | |
| --- | --- |
| Upstream | github/spec-kit |
| Fork base | markmals/speckit (renamed from `spec-kit` at fork time, per D1) |
| Pin commit | `1b0556c711b633a6d50b2e2f5f8db0e6717489d3` (2026-06-11 — "Update Linear Integration extension to v0.4.0 (#2942)") |
| Pin tag | `fork-base` — preserves the pinned Python CLI as the **Phase-2 oracle** even as `main` diverges |
| Upstream license | MIT (notice retained in `LICENSE`; fork copyright added) |
| Working branch | `main` (in-place rewrite) |

Diff against the oracle: `git show fork-base:src/specify_cli/<module>.py`, or stand up a worktree — `git worktree add ../speckit-oracle fork-base`.

## Identity (D1)

Retained, deliberately: product **SpecKit**, binary **`specify`**, slash namespace **`/speckit.*`**. The fork inherits upstream's command lineage rather than re-coining it; provenance rides on this file plus the retained license notice. Trade-off accepted: a near-identical name reads as a continuation of spec-kit, so the "not affiliated with GitHub" signal is carried by branding + this file, not by the name. Branding (logo, `newsletters/`, `media/`, `.zenodo.json`) is dropped/replaced — see the disposition map.

## Disposition map

Status: **rewrite** (port behavior to Go) · **replace** (new design supersedes it) · **drop** · **audit** (semantics need a source read before deciding port-vs-drop).

| Upstream asset | Fate | Notes |
| --- | --- | --- |
| `src/specify_cli/commands/` | rewrite (Go) | init, check, version, self-upgrade |
| `src/specify_cli/integrations/` — 32 agents + `generic` | rewrite 3 + AGENTS.md; drop 29 | Keep **claude, codex, copilot** (D4) + `generic` → shared AGENTS.md emitter. Drop: agy, amp, auggie, bob, cline, codebuddy, cursor_agent, devin, forge, gemini, goose, hermes, iflow, junie, kilocode, kimi, kiro_cli, lingma, opencode, pi, qodercli, qwen, roo, rovodev, shai, tabnine, trae, vibe, windsurf. |
| `integrations/` scaffolding — `base.py`, `manifest.py`, `catalog.py`, `_commands.py`, `_install_commands.py`, `_migrate_commands.py`, `_query_commands.py`, `_helpers.py` | rewrite (Go) | This **is** the projection/adapter system — port as the per-family adapter interface (D4) |
| `extensions.py`, `catalogs.py`, `integration_state.py`, `integration_status.py`, `integration_runtime.py`, `_agent_config.py`, `agents.py` | rewrite (Go) | install/catalog/state — fork owns the format; single `.speckit/` dotdir (D6) |
| `presets.py` | rewrite (Go) | keep mechanism; audit shipped presets |
| `src/specify_cli/workflows/` (16 files, incl. `steps/`) | **drop / defer** ⚠ | A ~1965-line YAML pipeline engine (`engine.py`, `catalog.py`, step types) that upstream `init` uses to install a built-in "speckit" workflow. The fork's lifecycle (D9 disposable tasks + the Mocha engine) supersedes pipeline orchestration — don't port it. Post-v1 feature at most. **Confirm with owner before deleting in Phase 2.** |
| `src/specify_cli/authentication/` (6 files) | **port-minimally (Phase 2+)** | Optional, separable; not used by `init`. When remote extension/preset install lands, port `http.py` (redirect-strip safety) + `config.py` (opt-in `auth.json`); drop the Azure DevOps OAuth flow. |
| `_assets.py`, `_github_http.py`, `_console.py`, `_utils.py`, `_init_options.py`, `shared_infra.py`, `_version.py` | rewrite (Go) | supporting infra; fold into Go packages |
| `templates/` | replace | Workbench per-kind spec templates; plan/tasks adapted per D9 |
| `scripts/bash/` + `scripts/powershell/` | rewrite → subcommands | logic → Go; only shell left is git-hook trampolines (D2); no `.specify/` compat shims (D6) |
| `extensions/` catalogs + publishing guide | replace | first-party catalog; community catalog snapshotted as design reference only (D6) |
| top-level `workflows/` (6 files) | **drop / defer** ⚠ | Static workflow catalog + bundled `speckit/workflow.yml` + docs (not `.github/workflows`). Tied to the workflow-engine decision above — if that drops, these go too (keep `README`/`ARCHITECTURE` as ancestry reference if useful). |
| `.github/` release pipeline | replace | goreleaser + Go template-package build |
| `docs/`, `spec-driven.md` | rewrite | fork identity; keep methodology essay as ancestry reference |
| `tests/` | mine | seed corpus for golden-tree tests |
| `AGENTS.md` (top level) | rewrite | the fork's own + the shared Codex/Copilot substrate (D4) |
| `newsletters/`, `media/`, `.zenodo.json` | drop | upstream branding |
| `CITATION.cff` | drop | academic citation for github/spec-kit; affiliation surface — FORK.md carries provenance instead |
| `pyproject.toml` | drop | replaced by Go module + goreleaser |

## Divergence log

Decisions that intentionally break from upstream (full rationale in `speckit-fork-plan.md` §2):

- **D2** — runtime is a present Go binary, not an init-time script installer; all `scripts/` logic moves into `specify` subcommands.
- **D3** — curated stacks are first-party bundled extensions; `init --platforms` installs them (replaces Workbench's superset-then-prune `/setup`).
- **D4** — 3 agent families (Claude Code, Codex, Copilot) across CLI/GUI/extension surfaces; `AGENTS.md` is the shared Codex/Copilot substrate, not a long-tail fallback.
- **D6** — fork clean: single `.speckit/` dotdir, no `.specify/`, no compat shims; upstream community catalog is reference-only.
- **D7** — drift via content-hash acknowledgment lock (sharded), not mtime.
- **D8** — enforcement in git/CI via `specify gate`, not agent hooks.
- **D9** — trunk-based spec library; `plan.md`/`tasks.md` are disposable per-(spec×platform) artifacts.
- **D11 / D12** — parity's `declared-deviation` is human-attested (gated for sign-off); the scenario-to-test join is a hard-failing per-language conformance surface.
- **D13** — v1 platforms: web + apple only; the rest are post-v1 additive extensions.
