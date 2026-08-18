# CI gating — required checks & branch protection

SpecKit does not write your CI. Your project brings its own workflow; the gate
is a **stack-neutral** building block you call from it. The gate runs only the
target's configured `command` (that's what `specify verify` does) plus the
specify checks — it assumes no task runner, no package manager, and no
per-stack quality job. Whatever static checks your project runs (formatting,
linting, type checks) are its own jobs, defined by you.

## The spec gate

The gate runs, in order:

1. `specify scan` — the spec library is well-formed.
2. `specify gate firewall --against <PR base> --format github` — **the firewall**:
   a scenario-tagged test that changed without its spec changing is blocked, and
   annotated inline on the offending test file in the PR's Files-changed view.
   This is the demo — *you cannot merge a test that silently drifted from its spec.*
3. `specify verify <target> --format github` — run the target's configured
   `command`, join each scenario to its test, lock what passes; an unjoinable
   scenario is annotated at its spec line and a dangling test binding at the
   test line.
4. `specify parity <target> --gate --format github` — every scenario must conform
   (a `suspect`/`drifted`/`missing` cell fails the check, annotated at its spec line).

`specify verify` **already runs the target's test suite** (that's how it joins
each scenario to its bound test), so the gate never needs a separate test job —
tests are never run twice.

`--format github` makes `specify` emit [GitHub Actions workflow-command
annotations](https://docs.github.com/actions/reference/workflow-commands-for-github-actions#setting-an-error-message),
so every gate failure — a drifted test, an unjoinable scenario, a dangling
binding — shows up as an inline error at its exact `file:line`, not just a red ✗.

> **`gate generated` / `gate scope` are git hooks, not PR checks.** `verify`
> legitimately rewrites the committed locks under `.speckit/lock/` on green, so a
> `gate generated` PR check would false-positive on every honest lock update; and
> `gate scope` validates a single commit subject, which is a commit-time concern.
> Wire both as local hooks (see the README), not in CI.

## The stack-neutral CI shape

Two ways to run the gate; both leave toolchain setup to you.

**The reusable workflow** — when the runner's default toolchain can run your
target's `command` (or the report is committed/prebuilt), the whole job is one
line:

```yaml
# .github/workflows/ci.yml — yours, written by you
on: { pull_request: {} }
jobs:
  verify:
    uses: markmals/speckit/.github/workflows/gate.yml@v0.2.0
    with:
      target: web
```

**The composite action** — when the target needs its own setup (a language
toolchain, dependency install), run the gate as a step inside your own job,
after your own setup steps. Check out with full history so the firewall can
diff against the PR base:

```yaml
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with: { fetch-depth: 0 }
      # …your project's own toolchain + dependency setup steps here…
      - uses: markmals/speckit/gate@v0.2.0
        with:
          target: web
```

The gate itself contains nothing stack-shaped: it installs `specify` (via Go),
then runs `scan` → firewall → `verify <target>` → `parity --gate`. Everything
the target needs to run its own `command` is the caller's business.

> **Pin the ref deliberately.** SpecKit is pre-1.0, so there is no floating
> `v1` to track — the examples pin the concrete release tag, and you bump it
> when you want the new version. The composite action installs `specify` with
> `go install …@<specify_version>`, whose default is that same tag; override the
> input to pin a different tag, a branch, or a SHA. To run against unreleased
> `main`, use `…/gate.yml@main` / `…/gate@main` and set `specify_version: main`.

## Make the check required (branch protection)

A check only gates a merge if it's **required**. The reusable-workflow job's
status check is named **`verify / verify`** (caller job `verify` / reusable job
`verify`). Provision a ruleset on the default branch that requires it, requires
a PR, and blocks force-pushes:

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
          { "context": "verify / verify" }
        ]
      } },
    { "type": "non_fast_forward" }
  ]
}
JSON
```

`gh api` fills in `{owner}/{repo}` from the current repo. Adjust
`required_approving_review_count` to taste; `~DEFAULT_BRANCH` targets whatever
the repo's default branch is. If your project has its own quality jobs, add
their contexts to `required_status_checks` alongside the gate. (When you run
the composite action inside your own job, the context is that job's own name
instead of `verify / verify`.)

## Branch-protection via the UI

Settings → Rules → Rulesets → New branch ruleset → target the default branch →
enable **Require a pull request before merging** and **Require status checks to
pass**, then add `verify / verify` (it appears in the picker after the workflow
has run at least once on a PR).
