# Using SpecKit with GitHub

The repo holds the **truth**; GitHub holds the **process**. Specs, scenarios,
locks, tests, and agent memory live in the repo and are what the engine verifies.
Work items and the PR machinery are *ephemeral coordination* — you could delete
the whole board and lose nothing the engine proves. That is the determinism
line: the engine (`scan`/`verify`/`lock`/`drift`/`cover`/`parity`/`gate`) is
repo-local and offline, so a board call that fails can never block a local
`verify`.

Everything on this page is **optional**. The engine is correct without any of
it — GitHub is the workflow shell on top, not a dependency of the proof.

The GitHub-facing side inherits `gh`'s auth (`gh auth token`), so there's no
token to plumb and no credentials to configure. Outward mutations confirm
first; pass `--yes` to skip the prompt in a script.

## PR gating

Run the spec gate as a **required status check** so nothing merges with a type
of drift the engine can catch. Your project writes its own `ci.yml`; the gate
is one job:

```yaml
verify:
  uses: markmals/speckit/.github/workflows/gate.yml@v1
  with:
    target: web
```

The reusable workflow installs `specify` and runs, in order: `scan` → the
test-edit firewall → `verify <target>` (which runs the target's configured
`command`) → `parity --gate`. A target that needs its own toolchain setup runs
the composite action `markmals/speckit/gate@v1` inside its own job instead,
after its own setup steps.

Each step runs `--format github`, so every failure — a test edited away from its
spec, an unjoinable scenario, a dangling test binding, a drifted parity cell — is
**annotated inline** on the offending `file:line` in the PR's Files-changed view,
not just a red ✗.

A check only gates a merge if it's **required**. Provision the ruleset with the
`gh api` recipe in **[../ci-gating.md](../ci-gating.md)** — it requires
`verify / verify`, requires a PR, and blocks force-pushes. That page also has
the full gate breakdown and the stack-neutral CI shape.

> The `gate scope` / `gate generated` checks are **git hooks, not PR checks** —
> `verify` legitimately rewrites the committed locks on green, so a `gate generated`
> PR check would false-positive on every honest lock update. Wire them locally:
>
> ```sh
> # .git/hooks/commit-msg
> specify gate scope --message "$1"
> # .git/hooks/pre-commit
> specify gate firewall && specify gate generated
> ```

## Work tracking on GitHub — the `github-projects` provider

`specify work` drives a pluggable work-tracking provider; the GitHub one puts
work items on a **Projects v2 board** as issues. Select it in
`.speckit/specs.json`:

```json
"work": { "provider": "github-projects", "project": 1, "owner": "acme" }
```

The verbs are the same five as every provider — `ready` / `create` / `claim` /
`move` / `list` — with the canonical states mapped onto the board's Status
columns (`--status-field` and repeatable `--column <state>=<Column>` remap
them). `claim` assigns you and moves the card in one step, and refuses an item
someone else already holds — multi-agent safe.

Defect intake lives here too: `specify work create "<title>" --type defect`.
A defect's durable form is a regression scenario plus a bound test in the repo;
the work item is intake, closed (moved to `done`) on a green `verify` — the
lock is the proof. There is deliberately no durable item↔scenario backlink:
GitHub's automatic cross-references (the fix PR mentioning the issue) are
breadcrumb enough, and the repo never depends on the board.

The `/speckit.taskstowork` agent command files a freshly authored feature's
task list as work items through whatever provider is configured — on this
provider, that seeds the board.

The full provider reference — states, types, the flags, and the three other
providers (`markdown`, `beads`, `none`) — is
**[../work-providers.md](../work-providers.md)**.

Spec-derived work needs **no board at all**: `specify drift <target>` and
`specify cover <spec-id>` surface un-implemented or drifted specs straight from
the repo, so a spec ID is already a stable, greppable work item offline.

## Next

- **[offline.md](offline.md)** — the offline engine these surfaces wrap; the
  source of truth nothing here can override.
- **[../design/github-integration.md](../design/github-integration.md)** — the
  design rationale: the determinism-line table and the gh-auth inheritance.
- **[../../README.md](../../README.md)** — the overview and the complete
  `specify` command reference.

Everything on this page is **optional**. The engine verifies your specs with no
GitHub at all — the PR gate and the Projects board are the coordination shell
you opt into on top of it.
