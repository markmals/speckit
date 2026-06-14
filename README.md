# SpecKit

> Spec-driven development for native, multiplatform apps. You write the behavior once as a spec; each target implements it in its own native stack; the `specify` tool keeps every implementation honest against the spec.

SpecKit is a single Go binary. You use it to scaffold a project, then to continuously check that each target's code still does what the specs say. Specs are the source of truth and live alongside the code; the native implementations (web, iOS, Android, …) are how that one spec gets realized on each target.

It is a rewrite of [github/spec-kit](https://github.com/github/spec-kit) in Go, with one important difference: the `specify` binary stays in your project and *is* the verification engine, rather than being a one-time installer.

There are two halves to working in SpecKit, and the README covers both:

1. **Your coding agent** drafts specs and writes the native code, driven by the `/speckit.*` commands that `init` installs.
2. **The `specify` CLI** checks that code against the specs — what's verified, what's drifted, what's covered — deterministically, in your terminal and in CI.

## Status

Implemented and tested on Linux, macOS, and Windows: project scaffolding (`init`), the full engine (`scan`, `verify`, `lock`, `drift`, `cover`, `parity`, `gate`), and stack scaffolding (`target add`/`target register`). Two stacks land green on `verify` out of the box: the **web** stack (TanStack Start + Vitest, UI from the [racket-ui](https://github.com/markmals/racket-ui) shadcn registry; data `convex`/`drizzle`/`none` × runtime `cloudflare`/`node`; `--with` clerk·tiptap·posthog·email·stripe) and the **go-service** stack (a Go HTTP daemon in `cmd/`, members composing into one repo-root `go.mod`; `--with` openapi·sqlite·client for a contract-first, persistent, trove-shaped service). `specify target register` onboards a member that already exists (no scaffolding) — the path for adopting SpecKit in an existing repo. In progress: a published release (and the Homebrew/Mise install that depends on it), the remaining stack scaffolds (Apple next), the `check`/`self upgrade`/`extension`/`preset` commands, and the `claude-pack` (lifecycle hooks and review subagents) and `github-pack` (a CI action, spec→issues, worktree helpers). Until a release is cut, install from source.

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

First scaffold the target. `target add` lays down a runnable starter on the recommended stack, registers the target in `.speckit/specs.json`, projects the stack's skill pack, and installs dependencies — resolving each package to its current version by running the package manager rather than hardcoding versions:

```sh
specify target add web --stack web   # TanStack Start + Vitest, green on `verify` immediately
```

It arrives green on purpose: the scaffold seeds one example spec, one bound test, and a `// SPEC:` pointer, so your agent extends a working spec→test→verify loop instead of wiring one from scratch. (`--dir` to place it, `--product` to label it, `--with <feature>` to add an add-on, `--no-install` to skip the install.)

Adopting SpecKit in a repo whose code already exists? `target register` records an existing member as a target — it writes no files and runs nothing, just adds the `.speckit/specs.json` entry (seeded from the stack's scaffold when there is one, or wired with `--format`/`--command`/`--report`/`--source`/`--bindings` flags):

```sh
specify target register api --stack go-service --dir cmd/api   # existing member → a verifiable target
```

Then have your agent plan and implement the feature natively, **tests first**, with each test bound to the scenario it proves:

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
   `format` is `junit` (Vitest, Gradle), `swift` (Swift Testing's event stream), or `gotest` (`go test -json`). See [docs/config.md](docs/config.md) for the full schema, the optional `product` label, and the `bindings` mode. Then:
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

The lock is **sharded per spec** (`.speckit/lock/<target>/<spec-id>.json`), so worktrees verifying different specs never collide in it. (Worktree setup is still manual; `specify work` drives the GitHub-side board — see below.)

### Pull requests

Open a PR per feature (or per target bring-up). Run the engine as **required status checks** so nothing merges with drift or broken parity:

- `specify scan` — the spec library is well-formed
- `specify verify <target>` — the implicated targets are green
- `specify parity <target> --gate` — every scenario conforms (a `suspect` or a `drifted` cell blocks the merge)

The pre-commit `gate` checks keep each commit honest before it's pushed (see below).

### Issues and Projects

GitHub holds the **process**; the repo holds the **truth**. Issues are *ephemeral defect intake* and Projects are an *ephemeral work board* — you could delete the whole board and lose nothing the engine verifies. The durable artifacts (specs, scenarios, locks) stay in the repo. The GitHub commands inherit `gh`'s auth (`gh auth token`), so there's no token to configure; outward actions confirm first (`--yes` to skip).

- **Issues = defect intake** (`specify issues`). A defect filed via the scaffolded `defect.yml` form becomes a regression scenario + a bound test; the issue closes on a green `verify` (the lock is the proof). `specify issues list|create|close`.
- **Projects = the work board** (`specify work`, Pillar 3, Beads-informed). A kanban where **"ready" is just a column**, not a computed field. `specify work ready` lists the actionable column; `specify work claim <issue>` assigns you and moves the card to In Progress (one atomic claim); `specify work move <issue> --to <column>`; `specify work discover --from <issue>` files a mid-task follow-up with `discovered-from` provenance. Column names are `--column`/`--status-field` flags; the defaults match the canonical board (Backlog → **Ready** → In Progress → On Hold → Cancelled → Closed, where **Ready** is the actionable column).
- **Spec-derived work still works offline:** `specify drift`/`cover` derive un-implemented or drifted specs straight from the repo, so a spec ID is already a stable, greppable work item — no board required.

Make the gate bite with `specify protect`, which provisions the branch-protection ruleset (require `quality` + `verify / verify`, require a PR, block force-pushes) via the GitHub API.

### Actions (CI/CD)

`specify target add` drops a `.github/workflows/ci.yml` into the project root. It runs **two parallel jobs on every PR**, both meant to be required status checks:

- **`quality`** — the target's fast static checks (`fmt:check` / `lint` / `typecheck`) via its mise tasks.
- **`verify`** — the spec gate: `scan` → the **test-edit firewall** → `verify <target>` → `parity --gate`. Because `verify` runs the test suite, tests live here only — never in `quality`.

The `verify` job is one line, delegating to SpecKit's reusable workflow (so the gate updates with the `@v1` tag, no re-scaffold):

```yaml
verify:
  uses: markmals/speckit/.github/workflows/gate.yml@v1
  with: { target: web, working_directory: apps/web }
```

The gate runs `specify gate firewall … --format github`, so a test edited away from its spec is **annotated inline** on the offending file in the PR — the same workflow-command mechanism `oxlint --format github` uses. Make the checks required (`quality` + `verify / verify`) with the branch-protection recipe in **[docs/ci-gating.md](docs/ci-gating.md)**.

Keep each commit honest locally with the `gate` checks as git hooks (these are commit-time, not PR checks — `verify` legitimately rewrites locks on green):

```sh
# .git/hooks/commit-msg
specify gate scope --message "$1"
# .git/hooks/pre-commit
specify gate firewall && specify gate generated
```

Deploys are optional and none are required. `specify deploy add <kind>` drops a `.github/workflows/deploy.yml` for a target (`cloudflare-workers-ssr`, `cloudflare-workers-spa`, `railway`, `github-pages-spa`) and records the manifest. Secrets are **1Password references** (`op://…`) in the manifest — never values — and `specify secrets sync` resolves them through your local `op` straight into GitHub Actions secrets (`gh secret set`) and the platform store (`wrangler secret put` / `railway variables`), never echoing or writing them to disk. (`CLOUDFLARE_ACCOUNT_ID` is a committed identifier in `wrangler.jsonc`, not a secret.)

## The `specify` command reference

Run `specify <command>`. Reporting commands print a styled summary by default and accept `--json` for machine-readable output (pipe it to `jq`). Commands that find problems (`scan`, `drift`, `verify`, `parity --gate`, `gate`) exit non-zero so they work in scripts and CI.

### Set up a project

| Command | What it does |
| --- | --- |
| `specify init [name] --integration <agent>` | Create a project wired for your agent (`claude`, `codex`, `copilot`, `generic`). `--here` sets up the current directory; `--force` merges into a non-empty one. |
| `specify target add <name> --stack <stack>` | Scaffold a runnable starter for a target on its stack, register it in `.speckit/specs.json`, project the stack's pack, and install deps (versions resolved by running the package manager). `--dir`, `--product`, `--with <feature>`, `--no-install`. |
| `specify target register <name> --stack <stack>` | Register an **existing** member as a target (no scaffolding, no install) — for adopting SpecKit in a repo that already has code. Seeds the verify wiring from the stack's scaffold, or set it with `--format`/`--command`/`--report`/`--source`/`--bindings`. `--dir`, `--product`. |

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
| `specify gate scope [subject]` | Check that a commit subject — given positionally, or read from a file with `--message <file>` (how a `commit-msg` hook passes it) — starts with a recognized scope. |

Each `gate` check takes `--against <ref>` (diff against a ref instead of the staged set) and `--format text\|json\|github`; `--format github` emits CI annotations on the offending file (see [docs/ci-gating.md](docs/ci-gating.md)).

### Work on GitHub (Issues, Projects, deploys)

These inherit `gh`'s auth (`gh auth token`); no token or config block. Outward actions confirm first (`--yes` skips). They never run in the offline engine path.

| Command | What it does |
| --- | --- |
| `specify issues list\|create\|close` | Defect intake (Pillar 2). List/open/close issues; close-on-green is the discipline (the lock is the proof). `--label`, `--type`, `--json`. |
| `specify work ready` | List the actionable column — the ready queue. `--project`, `--column`, `--status-field`, `--json`. |
| `specify work claim <issue#>` | Atomic claim: assign yourself + move the card to In Progress. `--project`, `--column`. |
| `specify work move <issue#> --to <column>` | Move a card to a column. `--project`. |
| `specify work discover --from <issue#> --title …` | File a mid-task follow-up issue with `discovered-from` provenance (label + #N backlink); `--project` also adds it to the board. |
| `specify deploy add <kind> [target]` | Add a deploy workflow + record the manifest. `--ci`/`--runtime NAME=op://…`, `--dir`, `--force`. |
| `specify secrets sync [target]` | Resolve the manifest's `op://` references and push them to GitHub Actions + the platform store. `--dry-run`, `--yes`. |
| `specify protect` | Provision the branch-protection ruleset (require the gate, require a PR, block force-push). Re-runnable. `--require`, `--reviews`. |

### Other

| Command | What it does |
| --- | --- |
| `specify version` · `specify help` | Print the version; show help for any command. |

A few commands are designed but not built yet: `extension`, `preset`, `apply`, `reconcile`, `ledger`, `bench`. They report intent if you run them. (`check` and `self upgrade` are specified but not yet wired up.)

## What `init` installs

| In the project | What it is |
| --- | --- |
| `/speckit.*` commands | The authoring/implementation prompts, projected for your agent — Claude skills under `.claude/skills/`, Codex/`generic` skills under `.agents/skills/`, Copilot under `.github/`. |
| Process-discipline skills | `test-driven-development` (RED/GREEN), `verification-before-completion`, `adversarial-review`, `systematic-debugging`, `implementing-a-spec`, `brainstorming-feature`, `writing-user-stories`, `managing-memory` — projected into the agent's skills dir (claude/codex/generic). |
| Review subagents (claude-pack) | `spec-reviewer`, `test-gap-finder`, `drift-hunter`, `handoff-builder`, `visual-verifier` — projected into `.claude/agents/` (Claude Code only). |
| Rules | `code-quality`, `commit-discipline`, `spec-conventions`, `enforcement-hierarchy` — the always-loaded conventions, projected into the agent's rules dir (`.claude/rules/` · `.agents/rules/` · `.github/rules/`) and referenced from the orientation file. |
| Project memory | A seed `MEMORY.md` index in the agent's `memory/` dir (`.claude/memory/` · `.agents/memory/` · `.github/memory/`) — committed, repo-local working knowledge the engine never reads. Loaded each session (Claude `@import`; a read-at-start directive for the others). Maintain it with the `managing-memory` skill. |
| `.speckit/` | The runtime: the constitution, spec/plan/tasks/checklist templates, and (after `verify`) the lock. No shell scripts. |
| Orientation file | `CLAUDE.md` / `AGENTS.md` / `.github/copilot-instructions.md` for the agent — wires in the rules and the memory index. |

Coming: a `triaging-defects` skill (reframed around Issues) and lifecycle **hooks** (format-on-edit, reconcile reminders).

## Guides

Deeper, kept-current walkthroughs live in [`docs/`](docs/). Two axes — pick your agent, then your workflow.

**Per harness** — what `init` projects for your agent, and how to drive the `/speckit.*` commands there:

- [Claude Code](docs/harnesses/claude.md) — user-invocable skills, native `@import` orientation, and the Claude-only review subagents.
- [Codex](docs/harnesses/codex.md) — the `AGENTS.md` projection under `.agents/` (byte-for-byte identical to the generic adapter).
- [Generic (AGENTS.md)](docs/harnesses/generic.md) — the portable fallback for any `AGENTS.md`-aware agent that isn't one of the named three.
- [GitHub Copilot](docs/harnesses/copilot.md) — everything under `.github/`, each command projected as both a chat-mode and a slash-prompt.

**By workflow** — the same engine, with or without GitHub:

- [Offline](docs/usage/offline.md) — the engine alone: `scan` / `verify` / `lock` / `drift` / `cover` / `parity` / `gate` plus git hooks, no network.
- [With GitHub](docs/usage/github.md) — the optional shell on top: PR gating, Issues, the Projects board, deploys, and secrets.

## Concepts

- **The lock.** `.speckit/lock/<target>/<spec-id>.json` holds the spec content hash last verified green, sharded per spec so parallel worktrees never conflict. `verify` is the only writer; drift is hash-mismatch-or-missing — never file timestamps (git doesn't preserve them).
- **The join.** The scenario↔test binding is declared in *source* (a `// SPEC:`/`// [scenario.id]` comment, a Swift trait, a Vitest title); outcomes come from the runner's report, matched by test identity. An unjoinable scenario or a dangling binding is always a hard error; an untagged test is too under the default `strict` bindings, or out of scope under `scoped` (for suites that mix scenario tests with plain unit tests — see [docs/config.md](docs/config.md)).
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
