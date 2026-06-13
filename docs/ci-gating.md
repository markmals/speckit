# CI gating — required checks & branch protection

`specify target add` drops a `.github/workflows/ci.yml` into the project root. It
runs **two parallel jobs on every PR**, both meant to be **required status
checks** so nothing merges with a type error, a lint failure, or spec/test drift:

| Job | What it runs | Tests? |
| --- | --- | --- |
| `quality` | the target's fast static checks via its mise tasks — `fmt:check`, `lint`, `typecheck` | no |
| `verify` | the SpecKit spec gate (below) | yes |

`specify verify` **already runs the target's test suite** (that's how it joins
each scenario to its bound test), so the tests live only in `verify` — they are
never run twice.

## The spec gate (the `verify` job)

The `verify` job delegates to SpecKit's reusable workflow, which installs
`specify`, sets up the target's toolchain, and runs, in order:

1. `specify scan` — the spec library is well-formed.
2. `specify gate firewall --against <PR base> --format github` — **the firewall**:
   a scenario-tagged test that changed without its spec changing is blocked, and
   annotated inline on the offending test file in the PR's Files-changed view.
   This is the demo — *you cannot merge a test that silently drifted from its spec.*
3. `specify verify <target> --format github` — run the tests, join each scenario
   to its test, lock what passes; an unjoinable scenario is annotated at its spec
   line and a dangling test binding at the test line.
4. `specify parity <target> --gate --format github` — every scenario must conform
   (a `suspect`/`drifted`/`missing` cell fails the check, annotated at its spec line).

`--format github` makes `specify` emit [GitHub Actions workflow-command
annotations](https://docs.github.com/actions/reference/workflow-commands-for-github-actions#setting-an-error-message)
(the same mechanism `oxlint --format github` uses), so every gate failure — a
drifted test, an unjoinable scenario, a dangling binding — shows up as an inline
error at its exact `file:line`, not just a red ✗.

> **`gate generated` / `gate scope` are git hooks, not PR checks.** `verify`
> legitimately rewrites the committed locks under `.speckit/lock/` on green, so a
> `gate generated` PR check would false-positive on every honest lock update; and
> `gate scope` validates a single commit subject, which is a commit-time concern.
> Wire both as local hooks (see the README), not in `ci.yml`.

## Using the workflow

The scaffolded `ci.yml` already wires this. The `verify` job is one line:

```yaml
verify:
  uses: markmals/speckit/.github/workflows/gate.yml@v1
  with:
    target: web
    working_directory: apps/web
```

Prefer the gate as a step inside your own job? Check out with full history, then
run the composite action directly:

```yaml
- uses: actions/checkout@v6
  with: { fetch-depth: 0 }
- uses: markmals/speckit/gate@v1
  with:
    target: web
    working_directory: apps/web
```

> **The `@v1` references are dormant until SpecKit's first release tag.** Both
> `go install …@v1` and `…/gate@v1` resolve only once `v1` is published. To try
> the gate before then, pin a branch or SHA: set the action's `specify_version`
> input and reference `…/gate.yml@main` / `…/gate@main`.

## Make the checks required (branch protection)

A check only gates a merge if it's **required**. The reusable-workflow job's
status check is named **`verify / verify`** (caller job `verify` / reusable job
`verify`); the static job is **`quality`**. Provision a ruleset on the default
branch that requires both, requires a PR, and blocks force-pushes:

```sh
gh api --method POST repos/{owner}/{repo}/rulesets --input - <<'JSON'
{
  "name": "speckit-gate",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "rules": [
    { "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 0,
        "dismiss_stale_reviews_on_push": false,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false
      } },
    { "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          { "context": "quality" },
          { "context": "verify / verify" }
        ]
      } },
    { "type": "non_fast_forward" }
  ]
}
JSON
```

`gh api` fills in `{owner}/{repo}` from the current repo. Adjust
`required_approving_review_count` to taste; `~DEFAULT_BRANCH` targets whatever the
repo's default branch is.

> Provisioning this from `specify` itself (it has `gh`'s token once it's a `gh`
> extension) is the planned automation; until then this `gh` recipe is the
> documented fallback.

## Branch-protection via the UI

Settings → Rules → Rulesets → New branch ruleset → target the default branch →
enable **Require a pull request before merging** and **Require status checks to
pass**, then add `quality` and `verify / verify` (they appear in the picker after
the workflow has run at least once on a PR).
