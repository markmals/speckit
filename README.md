# SpecKit

> Spec-driven development for native, multiplatform apps. One spec library is the contract; every platform implements it in its own native stack; the `specify` engine keeps them honest.

SpecKit is a hard fork of [github/spec-kit](https://github.com/github/spec-kit), rewritten in Go. Where upstream is an installer that's gone after `init`, SpecKit's binary **stays** — it's both the project bootstrapper and the verification engine (`scan` / `verify` / `drift` / `cover` / `parity` / `gate`). Specs live on `main` as the durable source of truth; native implementations are regeneration targets; drift is tracked deterministically by a content-hash lock.

See [`FORK.md`](FORK.md) for provenance and [`FORK-PLAN.md`](FORK-PLAN.md) for the full design (decisions D1–D15).

## Status

The **spec engine is implemented and CI-green** on Linux/macOS/Windows: `init` (four agent integrations) and `scan / verify / lock / drift / cover / parity / gate`. Not yet built: the meta commands (`check`, `self upgrade`), `extension`/`preset` management, the lifecycle commands (`apply`/`reconcile`/`ledger`), and the first-party platform packs + example repo that exercise `verify` against live web/apple toolchains. There is no published release yet — build from source.

## Build

```sh
go build -o specify ./cmd/specify
./specify version
```

(`mise use -g speckit`, Homebrew, and WinGet are planned.)

## How it works

- **Specs are the contract.** They live in `specs/` (cross-cutting) and `features/<NNNN>-<slug>/` (feature-scoped) as markdown with YAML frontmatter, dotted stable IDs, and Gherkin scenarios. See [`specs/CONVENTIONS.md`](specs/CONVENTIONS.md).
- **Each platform implements them natively** — TypeScript/React for web, Swift for Apple, Kotlin for Android, and so on. There is no shared code; the spec is what crosses platforms.
- **The binary is present at runtime.** The agent's slash-command prompts call `specify`, and `specify` is the engine that verifies implementations against scenarios and records what's green. No bash/PowerShell script layer.
- **Drift is a content hash, not a guess.** `specify verify` writes a per-(platform, spec) **lock** holding the spec content hash last verified green; `drift` reports any spec whose content changed since. Enforcement (`gate`) is agent-agnostic and runs in git hooks / CI.

## Commands

`specify <command> [flags]`. Every reporting command accepts `--json`. Findings-bearing commands (`scan`, `drift`, `verify`, `parity --gate`, `gate`) exit non-zero when they find something.

### Bootstrap

| Command | What it does |
| --- | --- |
| `specify init [name] --integration <agent>` | Scaffold a project, projecting the command set for the chosen agent (`claude`, `codex`, `copilot`, `generic`). `--here` uses the current directory; `--force` merges into a non-empty one. |

### Spec library

| Command | What it does |
| --- | --- |
| `specify scan [path]` | Lint the spec library against the model invariants **I1–I6** (filename↔id, closed kind, prefix, unique id, depends-on resolution, scenario sub-ids). Exits non-zero on findings. |
| `specify kinds` | List the closed spec-kind taxonomy. |

### Verification & tracking

| Command | What it does |
| --- | --- |
| `specify verify <platform> [path]` | Run the platform's tests (per `.speckit/verify/<platform>.json`), join outcomes to scenarios by **source-declared** bindings, and lock each spec whose scenarios all passed. Exits non-zero unless green. |
| `specify lock <platform> <spec-id>` | Acknowledge a spec green on a platform at its current content (normally invoked by `verify`). |
| `specify drift <platform> [path]` | Classify every spec on a platform: **drifted** (content changed since the lock), **missing** (never locked), or **clean**. Exits non-zero on drift. |
| `specify cover <spec-id> [path]` | Show one spec's state on each platform (conforming / drifted / missing), read from the lock without re-running tests. |
| `specify parity <platform> [path] [--gate]` | The five-state matrix, per scenario: **conforming · declared-deviation · drifted · suspect · missing**. `--gate` exits non-zero unless every cell is conforming. |

### Enforcement (D8) — for git hooks and CI

| Command | What it does |
| --- | --- |
| `specify gate firewall [--against <ref>]` | Fail if a scenario-tagged test changed without its owning spec also changing. |
| `specify gate generated [--against <ref>]` | Fail on edits to engine-generated paths (`.speckit/lock/`, codegen output). |
| `specify gate scope <subject>` (or `--message <file>`) | Validate a commit subject's scope against the defined scopes; reject the Conventional `type(scope):` form. |

### Utility

| Command | What it does |
| --- | --- |
| `specify version` · `specify help` | Binary version; Cobra-generated help for any command. |

### Planned (specced, not yet implemented)

`check`, `self upgrade`, `extension` (add/remove/search), `preset`, `apply`, `reconcile`, `ledger`, `work`, `bench`, `issues`. They are specified under `specs/` and `features/`, and report intent if invoked.

## The workflow for an app

The unit of work is **"satisfy this spec on this platform."** Specs are authored once and projected across platforms; the engine keeps the projections honest.

### 0 · Set up (once)

```sh
specify init my-app --integration claude
cd my-app
```

This writes the agent's command projection (e.g. `.claude/skills/speckit-*`), the `.speckit/` runtime (constitution, templates), and an orientation file. No runtime scripts — the prompts call `specify`.

### 1 · Author the spec (no code yet)

With your agent's projected `speckit-*` prompts, write the feature's specs — narrative → stories → models → scenarios — into `features/<NNNN>-<slug>/`. Leave open questions as `[NEEDS CLARIFICATION]`. Then lint:

```sh
specify scan          # exits non-zero on a malformed library
```

**The spec is the leverage — sharpen it here, before it forks across N platforms.**

### 2 · Implement on the reference platform (web first)

Have the agent implement each spec natively, **failing tests first**, with each test bound to its scenario *in source* — a `// [scenario.id]` comment, a Swift `.scenario("…")` trait, or a Vitest `it("[scenario.id] …")` title. Then:

```sh
specify verify web    # run the tests, join to scenarios, write the lock on green
specify drift web     # clean immediately after a green verify
```

A green `verify` means *the right scenarios were proven*, and records it. An unjoinable scenario, a dangling test reference, or an untagged test **fails loudly** (D12) — the join is source-declared, so the report never has to carry the scenario id (D15).

### 3 · Mirror to every other platform

Same specs, same order, once per platform. The web implementation stands beside the spec as a worked example:

```sh
specify verify apple
specify verify android
# …
```

### 4 · Keep it honest over time

```sh
specify cover <spec-id>     # which platforms implement a spec, and their state
specify drift <platform>    # what's gone stale since it was last verified green
specify parity <platform>   # per scenario: conforming / declared-deviation / drifted / suspect / missing
```

When a platform must diverge, annotate the implementation `// SPEC: <scenario-id> (deviates: <reason>)`. Parity shows it as `declared-deviation` (needs sign-off, never auto-green). A marker over a *failing* test is `suspect` — a marker can never hide a red test (D11).

### 5 · Enforce in CI and hooks

So honesty doesn't depend on anyone remembering:

```sh
specify gate scope --message "$1"   # commit-msg hook
specify gate firewall               # pre-commit: no untethered test edits
specify gate generated              # pre-commit: no hand-edited lock/codegen
specify parity web --gate           # CI: block the merge unless parity is clean
```

## Concepts

- **The lock (D7).** `.speckit/lock/<platform>/<spec-id>.json` holds the spec content hash last verified green, sharded per spec so parallel worktrees never conflict. `verify` is the only writer; drift is hash-mismatch-or-missing — never mtimes (git doesn't preserve them).
- **The join (D12 / D15).** The scenario↔test binding is declared in *source*; outcomes come from the runner's report, matched by test identity. Any unjoinable scenario or dangling/unbound test is a hard error.
- **Parity (D11).** Deviation-presence and test-outcome are crossed on **independent axes**, so a `(deviates:)` marker can't suppress a failing test.

## Layout & docs

| Path | What |
| --- | --- |
| [`FORK.md`](FORK.md) | Provenance, the upstream disposition map, the divergence log. |
| [`FORK-PLAN.md`](FORK-PLAN.md) | The full design — decisions D1–D15, phases, exit criteria. |
| [`specs/CONVENTIONS.md`](specs/CONVENTIONS.md) | The spec contract: IDs, kinds, frontmatter, reverse pointers, deviation markers, drift. |
| `cmd/specify/` | The Cobra CLI. |
| `internal/{specmodel,engine,reports,project,coreassets}` | The parser, the engine verbs, the report adapters, init, and embedded assets. |
| `specs/`, `features/` | The engine's own spec library — the project is its own first user (`specify scan` runs clean on it in CI). |

## License

MIT — a hard fork of [github/spec-kit](https://github.com/github/spec-kit) (MIT). Upstream's copyright notice is retained in [`LICENSE`](LICENSE) alongside the fork's.
