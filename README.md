# SpecKit

> Spec-driven development for native, multiplatform apps. You write the behavior once as a spec; each platform implements it in its own native stack; the `specify` tool keeps every implementation honest against the spec.

SpecKit is a single Go binary. You use it to scaffold a project, then to continuously check that each platform's code still does what the specs say. Specs are the source of truth and live alongside the code; the native implementations (web, iOS, Android, …) are how that one spec gets realized on each platform.

It is a rewrite of [github/spec-kit](https://github.com/github/spec-kit) in Go, with one important difference: the `specify` binary stays in your project and *is* the verification engine, rather than being a one-time installer.

## Status

The core is implemented and tested on Linux, macOS, and Windows: project scaffolding (`init`) and the engine (`scan`, `verify`, `lock`, `drift`, `cover`, `parity`, `gate`). Still in progress: the `check` and `self upgrade` commands, extension/preset management, and the ready-made platform setups that let `verify` drive a real web or Apple test suite out of the box. There's no published release yet — build from source.

## Build

```sh
go build -o specify ./cmd/specify
./specify version
```

## How it works

- **You describe behavior in specs.** Specs are markdown files with a little structured header, a stable ID, and acceptance scenarios written as plain Given/When/Then. They live in `specs/` (shared) and `features/<NNNN>-<slug>/` (per feature). See [the spec conventions](specs/CONVENTIONS.md).
- **Each platform implements the spec natively.** There's no shared runtime or cross-platform framework — web is built in React, Apple in Swift, Android in Kotlin, and so on. The spec is the only thing shared across them.
- **`specify` checks the implementations against the specs.** It runs each platform's tests, matches the results back to the scenarios they prove, and records which specs are genuinely passing on which platforms. From then on it can tell you what's changed, what's covered, and where a platform has drifted out of sync.

## Commands

Run `specify <command>`. Reporting commands accept `--json`. Commands that find problems (`scan`, `drift`, `verify`, `parity --gate`, `gate`) exit with a non-zero status so they work in scripts and CI.

### Set up a project

| Command | What it does |
| --- | --- |
| `specify init [name] --integration <agent>` | Create a new project wired for your coding agent (`claude`, `codex`, `copilot`, or `generic`). Use `--here` to set up the current directory, `--force` to merge into a non-empty one. |

### Work with the spec library

| Command | What it does |
| --- | --- |
| `specify scan [path]` | Check the spec library for problems — malformed IDs, duplicate IDs, broken cross-references, scenarios missing IDs, and the like. Reports each issue and exits non-zero if any are found. |
| `specify kinds` | List the kinds of spec the project understands (story, model, error, …). |

### Verify and track each platform

| Command | What it does |
| --- | --- |
| `specify verify <platform>` | Run that platform's tests, match the results to the scenarios they prove, and mark each fully-passing spec as verified for that platform. Exits non-zero unless everything it checked passed. |
| `specify lock <platform> <spec-id>` | Mark a spec as verified-good on a platform at its current contents (usually done for you by `verify`). |
| `specify drift <platform>` | List the specs whose text has changed since they were last verified on that platform (**drifted**), and those never verified there (**missing**). Exits non-zero if anything drifted. |
| `specify cover <spec-id>` | Show one spec's status on every platform — verified, drifted, or not implemented — at a glance. |
| `specify parity <platform> [--gate]` | Show every scenario's status on a platform: **conforming**, an intentional **declared-deviation**, **drifted**, **suspect** (more on this below), or **missing**. Add `--gate` to exit non-zero unless everything conforms. |

### Enforce in git hooks and CI

| Command | What it does |
| --- | --- |
| `specify gate firewall` | Block a change that edits a test tied to a scenario without also touching that scenario's spec — so tests can't be quietly weakened to pass. |
| `specify gate generated` | Block edits to files SpecKit generates and owns. |
| `specify gate scope <subject>` | Check that a commit message starts with a recognized scope. |

### Other

| Command | What it does |
| --- | --- |
| `specify version` · `specify help` | Print the version; show help for any command. |

A few commands are designed but not built yet: `check`, `self upgrade`, `extension`, `preset`, `apply`, `reconcile`, `ledger`, `work`, `bench`, and `issues`. They'll tell you so if you run them.

## Building an app, step by step

The job is always "make this spec true on this platform." You write a spec once and bring each platform up to it.

### 1. Create the project

```sh
specify init my-app --integration claude
cd my-app
```

You get your agent's commands installed, a place for specs, and the `specify` tool ready to use.

### 2. Write the spec — no code yet

Use your agent to draft the feature: the story, the data it touches, and the acceptance scenarios. Anything you're unsure of, leave marked for clarification and resolve it before moving on. Then check the library is well-formed:

```sh
specify scan
```

This is where the leverage is. The clearer the spec, the cleaner every platform that follows from it.

### 3. Build it on your first platform

Have your agent implement the spec — tests first — and make sure each test names the scenario it proves. Then verify:

```sh
specify verify web
specify drift web    # clean, right after a passing verify
```

A passing `verify` doesn't just mean "tests are green" — it means the *right* scenarios were actually proven. If a scenario has no test, or a test points at a scenario that doesn't exist, `verify` fails and tells you exactly which one.

### 4. Bring up the other platforms

Same specs, one platform at a time. The first implementation serves as a worked example:

```sh
specify verify apple
specify verify android
```

### 5. Keep everything honest over time

```sh
specify cover <spec-id>     # where a spec stands across platforms
specify drift <platform>    # what changed since it was last verified
specify parity <platform>   # the full per-scenario picture for a platform
```

When a platform genuinely needs to behave differently, note it in the code with a short reason. `parity` then shows that scenario as a **declared-deviation** instead of a failure — but if the test for it is actually failing, it shows up as **suspect** instead. Marking something as intentional can never hide a real failure.

### 6. Wire it into CI

So none of this depends on remembering:

```sh
specify gate scope --message "$1"   # commit-message hook
specify gate firewall               # pre-commit
specify gate generated              # pre-commit
specify parity web --gate           # CI: don't merge unless the platform is in sync
```

## Project layout

| Path | What's there |
| --- | --- |
| [`specs/CONVENTIONS.md`](specs/CONVENTIONS.md) | How specs are written — IDs, kinds, scenarios, and how code points back to them. Read this before writing specs. |
| `specs/`, `features/` | The spec library. (This repo specs *itself* — `specify scan` runs clean on it.) |
| `cmd/specify/`, `internal/` | The CLI and the engine. |

## License

MIT — a fork of [github/spec-kit](https://github.com/github/spec-kit) (MIT). Upstream's copyright notice is retained in [`LICENSE`](LICENSE) alongside the fork's.
