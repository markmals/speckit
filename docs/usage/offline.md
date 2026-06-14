# Using SpecKit offline

The engine is offline by construction. `scan`, `verify`, `lock`, `drift`,
`cover`, `parity`, and `gate` live in `internal/engine` + `internal/specmodel`,
and they never read GitHub or the network. Every line of GitHub/network code
lives in `internal/github`, which is imported only by the `cmd/specify` command
constructors — never by the engine. So the offline guarantee holds
*structurally*, not by convention: nothing GitHub is required for correctness,
and a board-sync or issue-call failure can never block a local `verify`.

That line is the spine of this whole document. Everything here runs from your
terminal against the repo on disk. No `gh`, no token, no config, no remote.

The truth lives in the repo: scenarios and acceptance criteria; the lock, drift,
and parity state under `.speckit/`; code and tests with their `// SPEC:`
pointers; agent memory in markdown. GitHub only ever holds ephemeral
*coordination* — defects, a work board, PR gating — and you could delete all of
it without losing anything the engine verifies. The GitHub workflow is the
[companion doc](github.md); this one is SpecKit with the network unplugged.

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
# 3. target add — register a target so verify knows how to run its tests
specify target add web --stack web   # green on verify out of the box

# …or, adopting SpecKit in a repo whose code already exists, register the member
# in place (writes no files, runs nothing) — seeded from the stack's scaffold, or
# wired explicitly when the member differs / its stack has no scaffold:
specify target register api --stack go-service --dir cmd/api
specify target register lib --stack ts-lib --dir packages/lib \
  --format junit --command "cd packages/lib && mise run test" \
  --report packages/lib/junit.xml --source packages/lib/src --bindings scoped
```

```text
# 4. implement on the target — tests first, each bound to its scenario
/speckit.plan      "Web: React + TanStack, tests in Vitest"
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

`command` is what runs the tests; `report` is where their results land; `format`
is `junit` (Vitest, Gradle) or `swift` (Swift Testing's event stream); `source`
is the directory scanned for the scenario↔test bindings. `stack` selects the
target's platform pack, and an optional `product` label groups targets. `scan`
validates this file whenever it's present; an absent one is fine — engine
commands that need a target just tell you to configure one. Full schema in
[../config.md](../config.md).

## Keep commits honest with git hooks

The `gate` checks run at commit time, locally, before anything is pushed. They
are **not** PR checks — `verify` legitimately rewrites the committed locks under
`.speckit/lock/` on green, so a `gate generated` check belongs at commit time,
not in CI. (The PR gate is in the [GitHub doc](github.md).)

| Check | What it blocks |
| --- | --- |
| `specify gate firewall` | A change that edits a scenario-tagged test without touching that scenario's spec. |
| `specify gate generated` | Edits to files SpecKit generates and owns (`.speckit/lock/`, codegen). |
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
Swift trait, a Vitest title), and the outcomes come from the runner's report,
matched by test identity. The two halves meet in `verify`. Anything that can't
be joined — a scenario with no test, a test naming a scenario that doesn't
exist — is a **hard error**, not a silent pass.

**Parity.** Deviation-presence and test-outcome are crossed on **independent
axes**. A `// SPEC: <id> (deviates: <reason>)` marker declares an intentional
difference, but it can never suppress a failing test: if the test is red, that
cell shows as **suspect**, not **declared-deviation**. Marking something
intentional cannot hide a real failure.

## Adopt on an existing repo

The engine works on an existing spec library with **no migration**. SpecKit uses
the same conventions as the Workbench template, so `specify scan` runs clean on a
Workbench-conventions project today. To adopt SpecKit offline:

```sh
# 1. Confirm the library is healthy
specify scan

# 2. After declaring your targets in .speckit/specs.json (above), verify one
specify verify web

# 3. Project the platform packs for your targets' stacks
specify packs
```

Then make sure each test names its scenario in source — that's the binding
`verify` joins on. If your existing tests already carry scenario tags, you're
done; otherwise add them as you verify each spec. You don't need `init` on an
existing project (it's for new ones); `init --here` can add the `/speckit.*`
command projections to a project that lacks them.

## Next

- [Working with GitHub](github.md) — add the GitHub workflow on top of this
  offline core: PR gating, Issues, Projects, deploys.
- [Project README](../../README.md) — the full command reference and project
  overview.
- [Spec conventions](../../specs/CONVENTIONS.md) — how specs are written, and
  how code points back to them.
