---
name: appkit-session-report
description: Use when the user wants to analyze, summarize, or diagnose the current or a recent Claude Code session for this macOS AppKit project — session feedback, where the agent got stuck, turn/token metrics, or a write-up to attach to a bug report. User-invoked.
disable-model-invocation: true
---

# AppKit Session Report

Produce a diagnostic for a Claude Code session working on **this** macOS AppKit project: metrics (turns, duration, tokens incl. cache), which skills loaded and when, the build success-vs-failure pattern across the Tuist/`xcodebuild` loop, where the agent got stuck, and a write-up suitable for a bug report. **User-invoked only** (`disable-model-invocation: true`) — don't load this on your own.

> **Don't hand-grep the transcript by eye.** A purpose-built analyzer, `analyze-session.swift`, ships in **mac-dev-skills** (`plugins/appkit/skills/appkit-session-report/`) — it reads the session JSONL under `~/.claude/projects/…`, classifies turns, and emits the report below. It is **not vendored into this pack**: SpecKit projects only the recipe. If the user has that plugin checked out, run its script; otherwise reconstruct the same shape by hand from the transcript.

## Run the analyzer (if available)

It is a single-file hashbang `swift` script (Foundation only — no packages); the Swift toolchain from Xcode is the only requirement. **Run it from the project directory and always pass `--output`:**

```bash
.../mac-dev-skills/plugins/appkit/skills/appkit-session-report/analyze-session.swift \
  --output session-report.md
```

```bash
# variants
.../analyze-session.swift --session-id <uuid> --output session-report.md
.../analyze-session.swift --events-file <transcript.jsonl> --output report.md
.../analyze-session.swift --skip-subagents --output session-report.md   # parent-only
```

Two reasons `--output` is non-negotiable:

1. A bare run prints the **entire unredacted report to stdout** — dumping verbatim prompts, paths, and any pasted secrets straight back into the agent's context. `--output` writes a file instead.
2. `--output` is what fires the script's stderr **privacy banner**.

The analyzer auto-selects the newest session whose recorded `cwd` matches where you run it — run it elsewhere and it can silently analyze an unrelated session. It is **Claude Code only** and honors `$CLAUDE_SESSION_ID`.

## Privacy — surface this every time

The report is your **unredacted** session: file contents the agent read/edited, your prompts verbatim (including pasted secrets), tool output, signing identities, and local `/Users/<you>/…` paths. When you report findings, restate this in your own words — don't let it stay buried in script output:

> ⚠️ **Before you share `session-report.md`** — it's your unredacted transcript. Read it end-to-end before attaching it to a public issue or posting it outside your org, and redact anything sensitive — or ask me for just the high-level metrics instead.

**Offer the summary, not the raw file.** If the user only needs metrics (turns, tokens, skills, build pattern, stuck patterns), summarize and share *that*, and say so. Hand over the raw file only after the user acknowledges the trade-off — this matters most for the "attach it to a GitHub issue" framing, exactly where an unredacted transcript leaks.

## Workflow

1. **Run the analyzer** (or reconstruct the shape by hand) with `--output session-report.md`, from the project dir.
2. **Summarize the key findings**: turns / duration / tokens (incl. cache), which skills loaded and when, build success-vs-failure pattern, stuck patterns, tooling friction.
3. **Surface the privacy reminder** above and offer summary-over-raw.
4. **Add your own observations** — was the app actually working? what's missing? AppKit-specific friction that would cut turns next time (a `mise run -C macOS …` rough edge, an `sdk-api`/`sdk-search` gap, a missing skill).

> **Filing a bug? Base claims on what the report shows, and verify before filing.** The report describes what the agent *did*, not what's *broken*. A `--help` probe or a verification command (`sdk-api check …`, `codesign --verify`) is **not** a build failure, and per-turn tool summaries are truncated command text, not proof of a defect. Confirm against the actual tool/file/skill before claiming a bug. See `systematic-debugging`.

## What the report covers

| Section | Details |
|---------|---------|
| Privacy and sensitivity | Unredacted-content warning, injected above the overview |
| Overview | Session ID, model, duration, parent turns, subagents, output + cache tokens |
| Prompt | The first user request (truncated) |
| Turn Breakdown | Turns + tokens by category (build-fix, code-edit, explore, subagent, …) |
| Skills | Which skills loaded, on which turn (incl. inside subagents) |
| Subagents | Per-agent type / turns / duration / description |
| Build Analysis | Attempts (success/fail), the toolchain used, build errors per turn |
| Stuck Patterns | Repeated file reads (≥3×), build loops (≥3 consecutive fails), repeated cleans |
| Turn Detail | Every turn with category, tokens, and tools — parent + each subagent |

Build detection in this stack is Tuist/Xcode-tuned: `mise run -C macOS build`, `tuist`, `xcodebuild`, `swift build`/`swift test` (the headless `Core` loop run by `specify verify`). Error extraction is Swift/Xcode-flavored (`error:`, `BUILD FAILED`, `linker command failed`, availability gates like `'NSGlassEffectView' is only available in macOS 26 or newer`).

## When to use

- The user explicitly asks for a session report, session feedback, or "what happened" in this session.
- The user wants a diagnostic to attach to a bug report — run the analyzer, then **lead with the privacy reminder and offer the summary.**

For the build/run loop the report describes, see `appkit-dev-workflow`; for runtime UI inspection, `appkit-app-inspector`. For verifying work is actually done before claiming it, `verification-before-completion`.
