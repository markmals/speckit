# Using SpecKit offline

The engine is offline by construction. `scan`, `verify`, `lock`, `drift`,
`cover`, `parity`, and `gate` live in `internal/engine` + `internal/specmodel`
(reading reports via `internal/reports` and config via `internal/config`), and
those packages import **neither the GitHub client nor any work provider** —
not directly, not transitively. Every line of network code lives in
`internal/github` and the work-provider adapters under `internal/work/`, which
are wired up only in the `cmd/specify` command constructors, never in the
engine. So the offline guarantee holds *structurally*, not by convention:
nothing networked is required for correctness, and a board call or GitHub
failure can never block a local `verify`.

That claim is enforced, not asserted. `TestEngineImportFirewall`
(`internal/engine/import_firewall_test.go`) walks the transitive import closure
of `internal/engine`, `internal/specmodel`, `internal/reports`, and
`internal/config` and fails if `internal/github` or any `internal/work` package
appears in it. It runs with the rest of the suite, so the wall cannot be
breached by a stray import that merely compiles.

That line is the spine of this whole document. Everything here runs from your
terminal against the repo on disk. No `gh`, no token, no remote.

The truth lives in the repo: scenarios and acceptance criteria; the lock,
drift, and parity state under `.speckit/`; code and tests with their `// SPEC:`
pointers; agent memory in markdown. Even work tracking is offline by default —
the `markdown` provider is a committed `WORK.md`, no network and no external
binary ([../work-providers.md](../work-providers.md)). The GitHub workflow is
the [companion doc](github.md); this one is SpecKit with the network unplugged.

## The offline loop

The job is always "make this spec true on this target." The loop is: scan the
library, author the feature with your agent, register the target, verify, then
track. A worked sequence (the `/speckit.*` steps run inside your agent; the
`specify …` steps run in your terminal):

```sh
# 1. scan — the spec library is well-formed before you build on it
specify scan
```

```text
# 2. author — your agent writes the specs (no code yet)
/speckit.specify   "Users can create, rename, and archive projects"
/speckit.clarify                 # resolve every [NEEDS CLARIFICATION] with you
/speckit.analyze                 # read-only: gaps, contradictions, broken refs
```

```sh
# 3. target add — register your existing code so verify knows how to run its
# tests and read their report (writes one .speckit/specs.json entry, nothing else)
specify target add web --dir apps/web --format junit \
  --report apps/web/report.junit.xml --source apps/web/src \
  --command "npm --prefix apps/web test" --bindings scoped
```

```text
# 4. implement on the target — tests first, each bound to its scenario
/speckit.plan
/speckit.tasks
/speckit.implement
```

```sh
# 5. verify — run the tests, join them to scenarios, lock what passes
specify verify web

# 6. track — drift, coverage, and per-scenario parity, all from the repo
specify drift web                # clean right after a green verify
specify cover story.projects.create
specify parity web
```

A passing `verify` is stronger than "tests are green": it means the *right*
scenarios were proven. A scenario with no test, or a test that names a scenario
which doesn't exist, fails `verify` and is named. None of these steps touch the
network.

## The engine commands

Each command prints a styled summary and accepts `--json` for machine-readable
output. Commands that find a problem exit non-zero, so they drop straight into
scripts and CI.

| Command | What it does | Exit |
| --- | --- | --- |
| `specify scan [path]` | Check the spec library — malformed/duplicate IDs, broken cross-references, scenarios missing IDs — and validate `.speckit/specs.json`. | non-zero if any problem is found |
| `specify verify <target>` | Run the target's tests (per its `.speckit/specs.json` entry), join results to the scenarios they prove, and lock each fully-passing spec. The only lock writer. | non-zero unless everything it checked passed |
| `specify lock <target> <spec-id>` | Mark a spec verified-good at its current contents (usually done for you by `verify`). | non-zero on failure |
| `specify drift <target>` | List specs whose text changed since last verified (**drifted**) or were never verified (**missing**). | non-zero on drift |
| `specify cover <spec-id>` | Show one spec's status on every target — conforming, drifted, or missing. | reporting only |
| `specify parity <target> [--gate]` | Per-scenario status: **conforming**, **declared-deviation**, **drifted**, **suspect**, **missing**. | with `--gate`, non-zero unless all conform |
| `specify gate <check>` | Commit-time integrity checks (firewall / generated / scope). | non-zero on a violation |

`verify` is the heart of it: it's the only thing that writes a lock, and a
passing run is a guarantee that each scenario it touched was proven by a real,
joinable test. An unjoinable scenario or a test naming a nonexistent scenario is
a hard error, not a warning.

## Configure a target

`verify` needs exactly one thing: the target's entry in `.speckit/specs.json`.
That's the wiring that tells the engine how to run a target's tests and where to
find their report and source bindings.

```json
{
  "version": 2,
  "agent": "claude",
  "targets": {
    "web": {
      "dir": "apps/web",
      "command": "npm --prefix apps/web test",
      "format": "junit",
      "report": "apps/web/report.junit.xml",
      "source": "apps/web/src"
    }
  }
}
```

`command` is what runs the tests; `report` is where their results land;
`format` is `junit` (any runner with a JUnit XML reporter), `swift` (Swift
Testing's event stream), or `gotest` (`go test -json`); `source` is the
directory (or directories) scanned for the scenario↔test bindings. `dir`
records where the target lives; an optional `product` label groups targets.
`scan` validates this file whenever it's present; an absent one is fine —
engine commands that need a target just tell you to configure one. Full schema
in [../config.md](../config.md); the flag-by-flag walkthrough (with a worked
example per report format) in [../adopting.md](../adopting.md).

## Keep commits honest with git hooks

The `gate` checks run at commit time, locally, before anything is pushed. They
are **not** PR checks — `verify` legitimately rewrites the committed locks under
`.speckit/lock/` on green, so a `gate generated` check belongs at commit time,
not in CI. (The PR gate is in the [GitHub doc](github.md).)

| Check | What it blocks |
| --- | --- |
| `specify gate firewall` | A change that edits a scenario-tagged test without touching that scenario's spec. |
| `specify gate generated` | Edits to files SpecKit generates and owns (`.speckit/lock/`). |
| `specify gate scope [subject]` | A commit subject that doesn't start with a recognized scope. |

Wire them as hooks:

```sh
# .git/hooks/commit-msg
specify gate scope --message "$1"

# .git/hooks/pre-commit
specify gate firewall && specify gate generated
```

`gate scope` reads the subject from a file with `--message <file>` (how the
`commit-msg` hook passes it). Every check takes `--against <ref>` to diff a ref
instead of the staged set, and `--format text|json|github`. `--format github`
emits CI annotations on the offending `file:line` — the same mechanism the PR
gate uses, but the hooks themselves run offline against your working tree.

## Concepts

**The lock.** `.speckit/lock/<target>/<spec-id>.json` records the spec content
hash last verified green. It's **sharded per spec**, so parallel worktrees
verifying different specs never collide in it. `verify` is the only writer.
Drift is detected by hash mismatch (or a missing lock), never by file
timestamps — git doesn't preserve those. The lock is the durable proof, and it's
just a file in the repo.

**The join.** The scenario↔test binding is declared in *source* (a comment, a
Swift trait, a test title), and the outcomes come from the runner's report,
matched by test identity. The two halves meet in `verify`. Anything that can't
be joined — a scenario with no test, a test naming a scenario that doesn't
exist — is a **hard error**, not a silent pass.

**Parity.** Deviation-presence and test-outcome are crossed on **independent
axes**. A `// SPEC: <id> (deviates: <reason>)` marker declares an intentional
difference, but it can never suppress a failing test: if the test is red, that
cell shows as **suspect**, not **declared-deviation**. Marking something
intentional cannot hide a real failure.

## Adopt on an existing repo

The engine works on an existing spec library with **no migration** — `specify
scan` runs clean on a Workbench-conventions project today. The full walkthrough
(one `target add` per existing implementation, a worked example per report
format, when to use `--bindings scoped` and `--reference`) is
[../adopting.md](../adopting.md).

## Next

- [Adopting SpecKit](../adopting.md) — register the project you already have.
- [Working with GitHub](github.md) — add the GitHub workflow on top of this
  offline core: PR gating and the Projects work provider.
- [Work providers](../work-providers.md) — the work-tracking surface, offline
  by default.
- [Project README](../../README.md) — the full command reference and project
  overview.
- [Spec conventions](../../specs/CONVENTIONS.md) — how specs are written, and
  how code points back to them.
