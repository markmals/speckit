# SpecKit

> Spec-driven development for native, multiplatform apps. You write the behavior once as a spec; each target implements it in its own native stack; the `specify` tool keeps every implementation honest against the spec.

SpecKit is a single Go binary. You use it to scaffold a project, then to continuously check that each target's code still does what the specs say. Specs are the source of truth and live alongside the code; the native implementations (web, iOS, Android, …) are how that one spec gets realized on each target.

It is a rewrite of [github/spec-kit](https://github.com/github/spec-kit) in Go, with one important difference: the `specify` binary stays in your project and *is* the verification engine, rather than being a one-time installer.

There are two halves to working in SpecKit, and the README covers both:

1. **Your coding agent** drafts specs and writes the native code, driven by the `/speckit.*` commands that `init` installs.
2. **The `specify` CLI** checks that code against the specs — what's verified, what's drifted, what's covered — deterministically, in your terminal and in CI.

## Status

Implemented and tested on Linux, macOS, and Windows: project scaffolding (`init`) and the full engine (`scan`, `verify`, `lock`, `drift`, `cover`, `parity`, `gate`). In progress: a published release (and the Homebrew/Mise install that depends on it), the `check`/`self upgrade`/`extension`/`preset` commands, the ready-made target setups that let `verify` drive a real web/Apple test suite out of the box, and the `claude-pack` (lifecycle hooks and review subagents) and `github-pack` (a CI action, spec→issues, worktree helpers). Until a release is cut, install from source.

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
- **Each target implements the spec natively.** No shared runtime or cross-platform framework — web in React, Apple in Swift, Android in Kotlin, and so on. The spec is the only thing shared across them.
- **Tests are bound to scenarios in the code.** Each test names the scenario it proves (a `// [scenario.id]` comment, a Swift `.scenario("…")` trait, a Vitest `it("[scenario.id] …")` title). That binding is read from source, not from test output.
- **`specify` checks implementations against specs.** It runs a target's tests, matches the results back to the scenarios they prove, and records which specs are genuinely passing on which target. From then on it tells you what's changed, what's covered, and where a target has drifted.

## Building an app, step by step

The job is always "make this spec true on this target." You author a feature once with your agent, build it on a reference target, then bring up the rest. Your agent's commands do the writing; the `specify` CLI does the checking.

> The `/speckit.*` commands below are installed into your agent by `init` (as Claude skills, Codex/Copilot commands, etc.). Run them in your agent; run `specify …` in your terminal.

### 1. Create the project

```sh
specify init my-app --integration claude   # or codex, copilot, generic
cd my-app
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

### 3. Build it on your first target (web)

Have your agent plan and implement the feature natively, **tests first**, with each test bound to the scenario it proves:

```text
/speckit.plan      "Web: React + TanStack, tests in Vitest"
/speckit.tasks                   # break the plan into ordered tasks
/speckit.implement               # write the failing tests, then the code to pass them
```

Then verify with the engine:

```sh
specify verify web               # run the tests, join to scenarios, lock what passes
specify drift web                # clean, right after a passing verify
```

A passing `verify` doesn't just mean "tests are green" — it means the *right* scenarios were proven. If a scenario has no test, or a test points at a scenario that doesn't exist, `verify` fails and names it.

### 4. Bring up the other targets

Same specs, one target at a time. The web implementation is a worked example the agent mirrors:

```text
/speckit.plan      "Apple: Swift + UIKit, tests in Swift Testing"
/speckit.implement
```
```sh
specify verify apple
```

### 5. Keep everything honest over time

```sh
specify cover <spec-id>     # where a spec stands across targets
specify drift <target>    # what changed since it was last verified
specify parity <target>   # the full per-scenario picture for a target
```

When a target genuinely must behave differently, note it in the code: `// SPEC: <scenario-id> (deviates: <reason>)`. `parity` shows that scenario as a **declared-deviation** instead of a failure — but if its test is actually failing, it shows up as **suspect**. Marking something intentional can never hide a real failure.

## Convert an existing project

Because SpecKit uses the same spec conventions as the Workbench template, **the engine works on an existing spec library with no migration** — `specify scan` runs clean on a Workbench project today. To adopt SpecKit:

1. **Install `specify`** (above), then from the project root confirm the library is healthy:
   ```sh
   specify scan
   ```
2. **Tell `verify` how to run each target's tests** by declaring your targets in `.speckit/specs.json`:
   ```json
   {
     "version": 1,
     "agent": "claude",
     "targets": {
       "web": {
         "stack": "web",
         "command": "pnpm -C apps/web test --run",
         "format": "junit",
         "report": "apps/web/report.junit.xml",
         "source": "apps/web/src"
       }
     }
   }
   ```
   `format` is `junit` (Vitest, Gradle) or `swift` (Swift Testing's event stream). See [docs/config.md](docs/config.md) for the full schema and the optional `product` label. Then:
   ```sh
   specify verify web
   ```
3. **Project the platform packs** for your targets' stacks — the stack-specific dev/verification skills:
   ```sh
   specify packs
   ```
4. **Make sure each test names its scenario** in source (the binding `verify` joins on). If your Workbench tests already carry scenario tags, you're done; otherwise add them as you verify each spec.

You don't need to run `init` on an existing project — it's for new projects. `init --here` can add the `/speckit.*` command projections to a project that doesn't have its own, but a Workbench project already ships its agent commands, so adopting SpecKit there is just the binary plus the targets in `.speckit/specs.json`.

## Working with Git and GitHub

SpecKit is **trunk-based**: the spec library lives on `main` as the durable source of truth, and implementation work happens on short-lived branches or worktrees that merge back. (GitHub is assumed throughout.)

### Branches and worktrees

The unit of work is "satisfy spec X on target Y." For parallel work — one agent on web while another does iOS — use a **git worktree per (spec × target)**:

```sh
git worktree add ../app-items-web feat/items-web
```

The lock is **sharded per spec** (`.speckit/lock/<target>/<spec-id>.json`), so worktrees verifying different specs never collide in it. (`specify work start <spec> <target>` will automate this — planned.)

### Pull requests

Open a PR per feature (or per target bring-up). Run the engine as **required status checks** so nothing merges with drift or broken parity:

- `specify scan` — the spec library is well-formed
- `specify verify <target>` — the implicated targets are green
- `specify parity <target> --gate` — every scenario conforms (a `suspect` or a `drifted` cell blocks the merge)

The pre-commit `gate` checks keep each commit honest before it's pushed (see below).

### Issues and Projects

You'll likely need fewer of these than usual: **the spec library is the source of work.** `specify drift` and `specify cover` derive the "ready" queue — specs that exist but aren't implemented, or that have drifted — straight from the repo, so a spec ID is already a stable, greppable work item. Use GitHub **Issues** for cross-cutting or non-spec work, and **Projects** for a board view if you want one; SpecKit doesn't require either. (Materializing specs/scenarios as issues via `specify issues` is planned.)

### Actions (CI/CD)

A few lines wire the engine into every PR:

```yaml
# .github/workflows/speckit.yml
name: speckit
on: pull_request
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: stable }
      - run: go install github.com/markmals/speckit/cmd/specify@latest
      - run: specify scan
      - run: specify verify web          # for each target you ship
      - run: specify parity web --gate
```

Add the `gate` checks to git hooks for fast local feedback:

```sh
# .git/hooks/commit-msg
specify gate scope --message "$1"
# .git/hooks/pre-commit
specify gate firewall && specify gate generated
```

A ready-made Action (the parity matrix as a check-run summary, spec→issues, the PR-comment-to-agent loop) is the planned **github-pack**; until then, the snippet above is all you need. Deploys run on merge to `main` however you like — SpecKit doesn't prescribe a target.

## The `specify` command reference

Run `specify <command>`. Reporting commands print a styled summary by default and accept `--json` for machine-readable output (pipe it to `jq`). Commands that find problems (`scan`, `drift`, `verify`, `parity --gate`, `gate`) exit non-zero so they work in scripts and CI.

### Set up a project

| Command | What it does |
| --- | --- |
| `specify init [name] --integration <agent>` | Create a project wired for your agent (`claude`, `codex`, `copilot`, `generic`). `--here` sets up the current directory; `--force` merges into a non-empty one. |

### Work with the spec library

| Command | What it does |
| --- | --- |
| `specify scan [path]` | Check the spec library for problems — malformed/duplicate IDs, broken cross-references, scenarios missing IDs — and validate `.speckit/specs.json`. Exits non-zero if any are found. |
| `specify packs [path]` | Project the platform skill packs for your targets' stacks (per `.speckit/specs.json`) into the agent's skills dir. |
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
| `specify gate generated` | Block edits to files SpecKit generates and owns (`.speckit/lock/`, codegen output). |
| `specify gate scope <subject>` | Check that a commit subject starts with a recognized scope. |

### Other

| Command | What it does |
| --- | --- |
| `specify version` · `specify help` | Print the version; show help for any command. |

A few commands are designed but not built yet: `check`, `self upgrade`, `extension`, `preset`, `apply`, `reconcile`, `ledger`, `work`, `bench`, `issues`. They report intent if you run them.

## What `init` installs

| In the project | What it is |
| --- | --- |
| `/speckit.*` commands | The authoring/implementation prompts, projected for your agent — Claude skills under `.claude/skills/`, Codex/`generic` skills under `.agents/skills/`, Copilot under `.github/`. |
| Process-discipline skills | `test-driven-development` (RED/GREEN), `verification-before-completion`, `adversarial-review` — the VSDD discipline, projected into the agent's skills dir (claude/codex/generic). |
| `.speckit/` | The runtime: the constitution, spec/plan/tasks/checklist templates, and (after `verify`) the lock. No shell scripts. |
| Orientation file | `CLAUDE.md` / `AGENTS.md` / `.github/copilot-instructions.md` for the agent. |

Coming (the rest of the process-pack / **claude-pack**): more skills (`systematic-debugging`, `triaging-defects`, `implementing-a-spec`), review **subagents** (`spec-reviewer`, `test-gap-finder`, `drift-hunter`), and lifecycle **hooks** (format-on-edit, reconcile reminders).

## Concepts

- **The lock.** `.speckit/lock/<target>/<spec-id>.json` holds the spec content hash last verified green, sharded per spec so parallel worktrees never conflict. `verify` is the only writer; drift is hash-mismatch-or-missing — never file timestamps (git doesn't preserve them).
- **The join.** The scenario↔test binding is declared in *source*; outcomes come from the runner's report, matched by test identity. Any unjoinable scenario or untagged test is a hard error.
- **Parity.** Deviation-presence and test-outcome are crossed on **independent axes**, so a `(deviates:)` marker can never suppress a failing test.

## Project layout

| Path | What's there |
| --- | --- |
| [`specs/CONVENTIONS.md`](specs/CONVENTIONS.md) | How specs are written — IDs, kinds, scenarios, and how code points back to them. Read this before writing specs. |
| `specs/`, `features/` | The spec library. (This repo specs *itself* — `specify scan` runs clean on it.) |
| `cmd/specify/`, `internal/` | The CLI and the engine. |
| [`FORK.md`](FORK.md), [`FORK-PLAN.md`](FORK-PLAN.md) | Provenance and the full design (decisions D1–D15). |

## License

MIT — a fork of [github/spec-kit](https://github.com/github/spec-kit) (MIT). Upstream's copyright notice is retained in [`LICENSE`](LICENSE) alongside the fork's.
