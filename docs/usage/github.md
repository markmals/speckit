# Using SpecKit with GitHub

The repo holds the **truth**; GitHub holds the **process**. Specs, scenarios,
locks, tests, and agent memory live in the repo and are what the engine verifies.
Issues, the project board, and the PR machinery are *ephemeral coordination* — you
could delete the whole board and lose nothing the engine proves. That is the
determinism line: the engine (`scan`/`verify`/`lock`/`drift`/`cover`/`parity`/`gate`)
is repo-local and offline, so a board-sync or issue call that fails can never block
a local `verify`.

Everything on this page is **optional**. The engine is correct without any of it —
GitHub is the workflow shell on top, not a dependency of the proof.

The GitHub-facing commands inherit `gh`'s auth (`gh auth token`), so there's no
token to plumb and no config block to write. Outward actions (creating issues,
moving cards, pushing secrets, provisioning rulesets) **confirm first**; pass
`--yes` to skip the prompt in a script.

## PR gating

`specify target add` drops a `.github/workflows/ci.yml` into the project root. It
runs **two parallel jobs on every PR**, both meant to be required status checks:

| Job | What it runs | Tests? |
| --- | --- | --- |
| `quality` | the target's fast static checks via its mise tasks — `fmt:check`, `lint`, `typecheck` | no |
| `verify` | the SpecKit spec gate (below) | yes |

`specify verify` **already runs the target's test suite** — that's how it joins
each scenario to the test that proves it — so the tests live in the `verify` job
**only**. They are never run twice; `quality` is just the static trio.

The `verify` job is one line, delegating to SpecKit's reusable workflow so the gate
updates with the `@v1` tag instead of a re-scaffold:

```yaml
verify:
  uses: markmals/speckit/.github/workflows/gate.yml@v1
  with:
    target: web
    working_directory: apps/web
```

The reusable workflow installs `specify`, sets up the target's toolchain, and runs,
in order: `scan` → the test-edit firewall → `verify <target>` → `parity --gate`.
Prefer it as a step inside your own job? The composite action
`markmals/speckit/gate@v1` runs the same gate after a full-history checkout.

Each step runs `--format github`, so every failure — a test edited away from its
spec, an unjoinable scenario, a dangling test binding, a drifted parity cell — is
**annotated inline** on the offending `file:line` in the PR's Files-changed view,
not just a red ✗. It's the same workflow-command mechanism `oxlint --format github`
uses.

A check only gates a merge if it's **required**. Run `specify protect` to provision
the branch-protection ruleset via the GitHub API — it requires `quality` and
`verify / verify`, requires a PR, and blocks force-pushes. It's re-runnable
(updates a same-named ruleset in place); `--require` and `--reviews` tune it. The
manual `gh api` ruleset recipe is the documented fallback.

See **[../ci-gating.md](../ci-gating.md)** for the full gate breakdown, the
annotation mechanism, and the branch-protection recipe.

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

## Issues — defect intake (Pillar 2)

Issues are *ephemeral defect intake*. A defect's durable form is a regression
scenario in the repo, not the issue.

```sh
specify issues list            # open defects
specify issues create          # file one
specify issues close <issue#>  # close on green
```

`--label`, `--type`, and `--json` are available. `specify target add` also drops the
`.github/` defect surface:

| File | Purpose |
| --- | --- |
| `ISSUE_TEMPLATE/defect.yml` | a defect form — stamps `type: Bug` plus a label fallback, prompts for repro + the target |
| `ISSUE_TEMPLATE/config.yml` | disables blank issues, links out to the SpecKit docs |
| `PULL_REQUEST_TEMPLATE.md` | a spec-touch checklist (specs changed? scenarios bound? `verify` green? `drift` clean?) |
| `CODEOWNERS` | routes `/features/`, `/specs/`, and `/.speckit/` to the spec owner, so **spec changes require human review** (replace `@OWNER` with a user or team) |
| `dependabot.yml` | for the stack's ecosystem (web → npm) |

The lifecycle is **scenario-canonical**:

```text
defect filed (via defect.yml — stamps type Bug + a label)
  → triage
  → the fix adds/updates a regression SCENARIO + a bound TEST in the repo
  → fix PR; merge → specify verify <target> joins the new scenario green → writes the lock
  → close the issue ON GREEN  (the lock is the proof; the issue was just intake)
```

There is deliberately **no durable issue↔scenario backlink** — GitHub's automatic
cross-references (the fix PR mentioning `#123`) are breadcrumb enough; the repo
never depends on the issue, so it can be closed, archived, or deleted freely.

To seed a board from a freshly authored feature, `specify taskstoissues` files that
feature's target task list as GitHub issues on the repo matching the git remote.
(It's the one `/speckit.*` command that touches GitHub.)

## Projects — the work board (Pillar 3)

The board is an *ephemeral* kanban, driven by the agent over an inlined Projects v2
GraphQL client (Projects is GraphQL-only). The defining choice: **"ready" is a
Status column, not a computed field.** You move a card into Ready; you don't ask the
engine to compute readiness.

```sh
specify work ready                         # list the actionable column
specify work claim <issue#>                # assign self + move to In Progress (one atomic mutation)
specify work move  <issue#> --to <column>  # move a card
specify work discover --from <issue#> --title …   # file a mid-task follow-up
```

`work claim` is a single atomic mutation (assign-self + column move), so it's
multi-agent safe. `work discover` files the follow-up with **`discovered-from:#N`
provenance** — a label plus a `#N` body backlink — and with `--project` also adds it
to the board.

Column names are **flags** (`--column` / `--status-field`) defaulting to the
canonical set:

```text
Backlog → Ready → In Progress → On Hold → Cancelled → Closed
```

Ready is the actionable column; On Hold is the blocked signal (the agent skips a
card showing blocked). The columns are flags, not a fixed schema — point them at
your board's own names if they differ.

Spec-derived work needs **no board at all**: `specify drift <target>` and
`specify cover <spec-id>` surface un-implemented or drifted specs straight from the
repo, so a spec ID is already a stable, greppable work item offline.

## Deploy & secrets

Deploys are optional, and **none are required** — the gate runs on PRs, deploys run
on push; they're independent. `specify deploy add <kind> [target]` drops a
`.github/workflows/deploy.yml` and records a **per-target** manifest:

| `<kind>` | Mechanism | Notes |
| --- | --- | --- |
| `cloudflare-workers-ssr` | `wrangler deploy` | SSR app on Workers (e.g. TanStack Start) |
| `cloudflare-workers-spa` | `wrangler deploy` with static assets | assets-only, no server worker |
| `railway` | Railway CLI in-workflow | server/container apps |
| `github-pages-spa` | `upload-pages-artifact` + `deploy-pages` | needs Pages enabled |

Workflows trigger on push to the default branch (`main`) plus a manual `workflow_dispatch`. `--ci NAME=op://…`,
`--runtime NAME=op://…`, `--dir`, and `--force` shape the manifest.

Secrets are **1Password references**, never values. The manifest holds only `op://`
pointers (raw values are rejected at `deploy add` and by `scan`):

```jsonc
"deploy": {
  "kind": "cloudflare-workers-ssr",
  "ci":      { "CLOUDFLARE_API_TOKEN": "op://Private/Cloudflare/api_token" },
  "runtime": { "DATABASE_URL": "op://Private/app-db/url" }
}
```

`specify secrets sync [target]` resolves each reference through your
locally-authenticated `op` and pushes it to **GitHub Actions** (`gh secret set`) and
the **platform store** (Cloudflare via `wrangler secret put` over stdin; Railway via
`railway variables`). Values are piped straight from `op` into `gh` / `wrangler` /
`railway` — **never echoed, logged, or written to disk**. `--dry-run` prints the
plan; `--yes` skips the confirm.

The `gh secret set` sync is the default. The optional upgrade is **runtime load**:
store a single `OP_SERVICE_ACCOUNT_TOKEN` in GitHub and have `deploy.yml` resolve the
`op://` references at deploy time via `1password/load-secrets-action`, so 1Password
is the only place each secret exists. `CLOUDFLARE_ACCOUNT_ID` is **not** a secret —
it's a committed identifier in `wrangler.jsonc`. (`op` is pinned alongside `gh`
in `mise.toml`.)

## Next

- **[offline.md](offline.md)** — the offline engine these commands wrap; the source
  of truth nothing here can override.
- **[../design/github-integration.md](../design/github-integration.md)** — the full
  design: the determinism-line table, the three pillars, deploy and secrets.
- **[../../README.md](../../README.md)** — the overview and the complete `specify`
  command reference.

Every command on this page is **optional**. The engine verifies your specs with no
GitHub at all — Issues, Projects, PRs, and deploys are the coordination shell you
opt into on top of it.
