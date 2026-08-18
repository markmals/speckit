# SpecKit

> A stack-agnostic spec engine that verifies behavior. You write the behavior once as a spec; each target implements it in whatever stack it already uses; the `specify` tool keeps every implementation honest against the spec.

SpecKit is a single Go binary. It adopts into a project that already exists — it generates no code and prescribes no stack. What it brings: a **spec library** with stable IDs, a **scenario↔test join** read from source, a **content-hash acknowledgment lock**, **drift/cover/parity** reporting, and **agent-agnostic gates**. Specs are the source of truth and live alongside the code; the native implementations (a web app, an iOS app, a Go daemon, …) are how that one spec gets realized on each target.

It is a rewrite of [github/spec-kit](https://github.com/github/spec-kit) in Go, with one important difference: the `specify` binary stays in your project and *is* the verification engine, rather than being a one-time installer.

There are two halves to working in SpecKit, and the README covers both:

1. **Your coding agent** drafts specs and writes the code, driven by the `/speckit.*` commands that `init` installs.
2. **The `specify` CLI** checks that code against the specs — what's verified, what's drifted, what's covered — deterministically, in your terminal and in CI.

## Status

Implemented and tested on Linux, macOS, and Windows: agent projection (`init`), target registration (`target add`), the full engine (`scan`, `verify`, `lock`, `drift`, `cover`, `parity`, `gate`), and pluggable work tracking (`work` over the `markdown`, `beads`, `github-projects`, and `none` providers). Three report formats are supported (`junit`, `swift`, `gotest`) — see [docs/report-formats.md](docs/report-formats.md) for how to add one.

## Install

Once a release is published:

```sh
# Homebrew
brew install markmals/tap/specify
```

```toml
# Mise — in your mise.toml
[plugins]
specify = "https://github.com/markmals/speckit"

[tools]
specify = "latest"
```

Or build from source today:

```sh
go install github.com/markmals/speckit/cmd/specify@latest
specify version
```

## How it works

- **You describe behavior in specs.** Markdown files with a small structured header, a stable ID, and acceptance scenarios written as plain Given/When/Then. They live in `specs/` (shared) and `features/<NNNN>-<slug>/` (per feature). See [the spec conventions](specs/CONVENTIONS.md).
- **Each target implements the spec natively.** No shared runtime or cross-platform framework — the web target in its web stack, the Apple target in Swift, the daemon in Go, whatever the project already is. The spec is the only thing shared across them.
- **Tests are bound to scenarios in the code.** Each test names the scenario it proves (a `// [scenario.id]` comment, a Swift `.scenario("…")` trait, a test title leading with `[scenario.id]`). That binding is read from source, not from test output.
- **`specify` checks implementations against specs.** It runs a target's tests, matches the results back to the scenarios they prove, and records which specs are genuinely passing on which target. From then on it tells you what's changed, what's covered, and where a target has drifted.

## Quickstart

```sh
specify init --here --integration claude   # wire your agent (or codex, copilot, generic)
specify target add web --dir apps/web --format junit \
  --report apps/web/report.junit.xml --source apps/web/src \
  --command "npm --prefix apps/web test" --bindings scoped
specify scan                               # the spec library + config are well-formed
specify verify web                         # run the tests, join scenarios, lock what passes
```

No code is generated and no platform is chosen at any point. The full
walkthrough — one worked example per report format — is
**[docs/adopting.md](docs/adopting.md)**.

## The loop, step by step

The job is always "make this spec true on this target." Your agent's commands do the writing; the `specify` CLI does the checking.

> The `/speckit.*` commands below are installed into your agent by `init` (as Claude skills, Codex/Copilot commands, etc.). Run them in your agent; run `specify …` in your terminal.

### 1. Wire up your project

```sh
specify init my-app --integration claude   # or codex, copilot, generic
cd my-app
# …or, in the project you already have:
specify init --here --integration claude
```

You get the `/speckit.*` commands wired for your agent, a `.speckit/` runtime, and a place for specs. The first time, set your project's ground rules:

```text
/speckit.constitution    # the principles every spec and target must honor
```

### 2. Author the feature — specs first, no code yet

Work with your agent to turn an idea into specs:

```text
/speckit.specify   "Users can create, rename, and archive projects"
/speckit.clarify                 # resolve every [NEEDS CLARIFICATION] with you
/speckit.analyze                 # read-only: gaps, contradictions, broken references
```

Then confirm the library is well-formed:

```sh
specify scan       # exits non-zero on a malformed spec library
```

This is where the leverage is — the clearer the spec, the cleaner every target that follows from it.

### 3. Register a target

A target is code you own — SpecKit writes nothing into it. `target add` records one entry in `.speckit/specs.json`: where the target lives, how to run its tests, and how to read the resulting report:

```sh
specify target add web --dir apps/web --format junit \
  --report apps/web/report.junit.xml --source apps/web/src \
  --command "npm --prefix apps/web test" --bindings scoped
```

`--bindings scoped` lets a suite with pre-existing plain unit tests verify the scenarios it does bind; `--reference` marks the target the others match in a multi-target repo. Worked examples for every report format are in **[docs/adopting.md](docs/adopting.md)**.

Then have your agent plan and implement the feature, **tests first**, with each test bound to the scenario it proves:

```text
/speckit.plan
/speckit.tasks                   # break the plan into ordered tasks
/speckit.implement               # write the failing tests, then the code to pass them
```

### 4. Verify with the engine

```sh
specify verify web               # run the tests, join to scenarios, lock what passes
specify drift web                # clean, right after a passing verify
```

A passing `verify` doesn't just mean "tests are green" — it means the *right* scenarios were proven. If a scenario has no test, or a test points at a scenario that doesn't exist, `verify` fails and names it.

### 5. Bring up the other targets

Same specs, one target at a time. The reference target's implementation is a worked example the agent mirrors:

```sh
specify target add ios --dir apps/ios --format swift … --bindings scoped
specify verify ios
```

### 6. Keep everything honest over time

```sh
specify cover <spec-id>   # where a spec stands across targets
specify drift <target>    # what changed since it was last verified
specify parity <target>   # the full per-scenario picture for a target
```

When a target genuinely must behave differently, note it in the code: `// SPEC: <scenario-id> (deviates: <reason>)`. `parity` shows that scenario as a **declared-deviation** instead of a failure — but if its test is actually failing, it shows up as **suspect**. Marking something intentional can never hide a real failure.

## Track the work

`specify work` is a small work-tracking surface over a pluggable provider — by default a committed `WORK.md` (no network, no external binary), or [Beads](https://github.com/steveyegge/beads), a GitHub Projects v2 board, or nothing at all:

```sh
specify work create "Bring the daemon target green" --spec story.adoption.target-add
specify work ready               # what's actionable
specify work claim wk-3          # take it → in-progress
specify work move wk-3 done
specify work list --state done
```

The engine never touches the provider — `scan`/`verify`/`drift`/`cover`/`parity`/`gate` run identically with or without one. Full reference: **[docs/work-providers.md](docs/work-providers.md)**.

## Working with Git and GitHub

SpecKit is **trunk-based**: the spec library lives on `main` as the durable source of truth, and implementation work happens on short-lived branches or worktrees that merge back.

The unit of work is "satisfy spec X on target Y." For parallel work, use a **git worktree per (spec × target)** — the lock is **sharded per spec** (`.speckit/lock/<target>/<spec-id>.json`), so worktrees verifying different specs never collide in it.

Run the engine as a **required status check** so nothing merges with drift or broken parity. Your project writes its own CI; the gate is one job calling the reusable workflow (or the composite action inside your own job):

```yaml
verify:
  uses: markmals/speckit/.github/workflows/gate.yml@v0.2.0
  with: { target: web }
```

The gate runs `scan` → the **test-edit firewall** → `verify <target>` → `parity --gate`, each with `--format github` so failures annotate the exact `file:line` in the PR. It is stack-neutral: it runs the target's configured `command` and the specify checks, nothing else. Branch-protection recipe and the full breakdown: **[docs/ci-gating.md](docs/ci-gating.md)**.

Keep each commit honest locally with the `gate` checks as git hooks (these are commit-time, not PR checks — `verify` legitimately rewrites locks on green):

```sh
# .git/hooks/commit-msg
specify gate scope --message "$1"
# .git/hooks/pre-commit
specify gate firewall && specify gate generated
```

## The `specify` command reference

Run `specify <command>`. Reporting commands print a styled summary by default and accept `--json` for machine-readable output (pipe it to `jq`). Commands that find problems (`scan`, `drift`, `verify`, `parity --gate`, `gate`) exit non-zero so they work in scripts and CI.

### Set up a project

| Command | What it does |
| --- | --- |
| `specify init [name] --integration <agent>` | Project the `/speckit.*` commands, skills, and rules for your agent (`claude`, `codex`, `copilot`, `generic`). `--here` sets up the current directory; `--force` merges into a non-empty one. |
| `specify target add <name> --dir <path> --format <junit\|swift\|gotest> --report <path> --source <path> [--source <path>…]` | Register existing code as a verifiable target — one `.speckit/specs.json` entry, no files generated. `--command <shell>` runs the tests; `--bindings strict\|scoped` sets how untagged tests are treated; `--product <label>` groups targets; `--reference` marks the reference target. |

### Work with the spec library

| Command | What it does |
| --- | --- |
| `specify scan [path]` | Check the spec library for problems — malformed/duplicate IDs, broken cross-references, scenarios missing IDs — and validate `.speckit/specs.json`. Exits non-zero if any are found. |
| `specify kinds` | List the kinds of spec the project understands (story, model, error, …). |

### Verify and track each target

| Command | What it does |
| --- | --- |
| `specify verify <target>` | Run the target's tests (per its entry in `.speckit/specs.json`), match results to the scenarios they prove, and lock each fully-passing spec. Exits non-zero unless everything it checked passed. |
| `specify lock <target> <spec-id>` | Mark a spec verified-good on a target at its current contents (usually done for you by `verify`). |
| `specify drift <target>` | List specs whose text changed since they were last verified (**drifted**) or were never verified (**missing**). Exits non-zero on drift. |
| `specify cover <spec-id>` | Show one spec's status on every target — conforming, drifted, or missing. |
| `specify parity <target> [--gate]` | Per-scenario status: **conforming**, **declared-deviation**, **drifted**, **suspect**, or **missing**. `--gate` exits non-zero unless everything conforms. |

### Enforce in git hooks and CI

| Command | What it does |
| --- | --- |
| `specify gate firewall` | Block a change that edits a scenario-tagged test without touching that scenario's spec. |
| `specify gate generated` | Block edits to files SpecKit generates and owns (`.speckit/lock/`). |
| `specify gate scope [subject]` | Check that a commit subject — given positionally, or read from a file with `--message <file>` (how a `commit-msg` hook passes it) — starts with a recognized scope. |

Each `gate` check takes `--against <ref>` (diff against a ref instead of the staged set) and `--format text\|json\|github`; `--format github` emits CI annotations on the offending file (see [docs/ci-gating.md](docs/ci-gating.md)).

### Track work

The provider comes from the `work` block in `.speckit/specs.json` (`markdown` by default; `beads`, `github-projects`, `none`). All verbs take `--json`. See [docs/work-providers.md](docs/work-providers.md).

| Command | What it does |
| --- | --- |
| `specify work ready` | List the actionable items — everything in `ready`. |
| `specify work create <title> [--type task\|defect] [--spec <spec-id>]` | File a work item; it lands in `ready`. |
| `specify work claim <id>` | Take an item — it moves to `in-progress`. |
| `specify work move <id> <state>` | Move an item (`ready`, `in-progress`, `blocked`, `done`). |
| `specify work list [--state <state>]` | List items, optionally filtered by state. |

### Other

| Command | What it does |
| --- | --- |
| `specify version` · `specify help` | Print the version; show help for any command. |

A few commands are designed but not built yet: `extension`, `preset`, `apply`, `reconcile`, `ledger`, `bench`. They report intent if you run them.

## What `init` installs

| In the project | What it is |
| --- | --- |
| `/speckit.*` commands | The authoring/implementation prompts, projected for your agent — Claude skills under `.claude/skills/`, Codex/`generic` skills under `.agents/skills/`, Copilot under `.github/`. |
| Process-discipline skills | `test-driven-development` (RED/GREEN), `verification-before-completion`, `adversarial-review`, `systematic-debugging`, `implementing-a-spec`, `brainstorming-feature`, `writing-user-stories`, `managing-memory` — projected into the agent's skills dir. |
| Review subagents (Claude only) | `spec-reviewer`, `test-gap-finder`, `drift-hunter`, `handoff-builder`, `visual-verifier` — projected into `.claude/agents/`. |
| Rules | `code-quality`, `commit-discipline`, `spec-conventions`, `enforcement-hierarchy` — the always-loaded conventions, projected into the agent's rules dir (`.claude/rules/` · `.agents/rules/` · `.github/rules/`) and referenced from the orientation file. |
| Project memory | A seed `MEMORY.md` index in the agent's `memory/` dir (`.claude/memory/` · `.agents/memory/` · `.github/memory/`) — committed, repo-local working knowledge the engine never reads. Maintain it with the `managing-memory` skill. |
| `.speckit/` | The runtime: the constitution, spec/plan/tasks/checklist templates, and (after `verify`) the lock. No shell scripts. |
| Orientation file | `CLAUDE.md` / `AGENTS.md` / `.github/copilot-instructions.md` for the agent — wires in the rules and the memory index. |

## Guides

Deeper, kept-current walkthroughs live in [`docs/`](docs/).

**Per harness** — what `init` projects for your agent, and how to drive the `/speckit.*` commands there:

- [Claude Code](docs/harnesses/claude.md) — user-invocable skills, native `@import` orientation, and the Claude-only review subagents.
- [Codex](docs/harnesses/codex.md) — the `AGENTS.md` projection under `.agents/` (byte-for-byte identical to the generic adapter).
- [Generic (AGENTS.md)](docs/harnesses/generic.md) — the portable fallback for any `AGENTS.md`-aware agent that isn't one of the named three.
- [GitHub Copilot](docs/harnesses/copilot.md) — everything under `.github/`, each command projected as both a chat-mode and a slash-prompt.

**By workflow:**

- [Adopting SpecKit](docs/adopting.md) — register the project you already have: one `target add` per implementation, worked examples per report format.
- [Offline](docs/usage/offline.md) — the engine alone: `scan` / `verify` / `lock` / `drift` / `cover` / `parity` / `gate` plus git hooks, no network.
- [With GitHub](docs/usage/github.md) — the optional shell on top: PR gating and the `github-projects` work provider.
- [Work providers](docs/work-providers.md) — the `specify work` surface and its four backends.
- [Config](docs/config.md) — the `.speckit/specs.json` schema.
- [Report formats](docs/report-formats.md) — how the engine reads test reports, and how to add a format.

## Concepts

- **The lock.** `.speckit/lock/<target>/<spec-id>.json` holds the spec content hash last verified green, sharded per spec so parallel worktrees never conflict. `verify` is the only writer; drift is hash-mismatch-or-missing — never file timestamps (git doesn't preserve them).
- **The join.** The scenario↔test binding is declared in *source* (a `// SPEC:`/`// [scenario.id]` comment, a Swift trait, a test title); outcomes come from the runner's report, matched by test identity. An unjoinable scenario or a dangling binding is always a hard error; an untagged test is too under the default `strict` bindings, or out of scope under `scoped` (for suites that mix scenario tests with plain unit tests — see [docs/config.md](docs/config.md)).
- **Parity.** Deviation-presence and test-outcome are crossed on **independent axes**, so a `(deviates:)` marker can never suppress a failing test.

## Project layout

| Path | What's there |
| --- | --- |
| [`specs/CONVENTIONS.md`](specs/CONVENTIONS.md) | How specs are written — IDs, kinds, scenarios, and how code points back to them. Read this before writing specs. |
| `specs/`, `features/` | The spec library. (This repo specs *itself* — `specify scan` runs clean on it.) |
| `cmd/specify/`, `internal/` | The CLI and the engine. |
| [`MIGRATION.md`](MIGRATION.md) | Migrating a project from the pre-stack-agnostic layout. |
| [`FORK.md`](FORK.md), [`FORK-PLAN.md`](FORK-PLAN.md) | Provenance and the original fork design (decisions D1–D15). |

## License

MIT — a fork of [github/spec-kit](https://github.com/github/spec-kit) (MIT). Upstream's copyright notice is retained in [`LICENSE`](LICENSE) alongside the fork's.
