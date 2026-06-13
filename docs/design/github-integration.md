# Design — GitHub-native integration

**Status:** proposal / direction-setting. The pivot from "Claude-native"
(Workbench) to **GitHub-native**: SpecKit stays agent-native and agent-agnostic
(Claude Code / Codex / Copilot), but its *banner* becomes that the framework
*lives on GitHub* — PRs gate, Issues hold defects, Projects hold the agent's work —
instead of being kept as repo markdown. Research current as of 2026-06.

## Thesis

> The repo holds the **truth**; GitHub holds the **process**; the engine projects one onto the other.

Two layers, joined by `specify`:

- A **portable spec-integrity core** — git + files + test runners; the engine
  (`scan`/`verify`/`lock`/`drift`/`cover`/`parity`/`gate`); works offline; never
  needs GitHub for *correctness*. This is the guarantee.
- A **GitHub-native workflow shell** — Issues / Projects / PRs / Actions. This is
  the banner.

The mechanic is **projection** — the same move we use for code. Code is a
materialized view of specs; GitHub surfaces are materialized views of *process*.
The difference from the spec library: **GitHub state is ephemeral coordination**,
not durable truth. The durable artifacts (specs, scenarios, locks, tests) live in
the repo; the issues and project cards are throwaway scaffolding around the work.

## The determinism line (what may move, what may not)

The test: **does the engine have to verify, hash, or diff this at a specific commit?**

| Artifact | Stays in repo? | Why |
| --- | --- | --- |
| Scenarios / acceptance criteria / models | **yes** | the join hashes & diffs them per commit; the BDD source |
| Locks, parity, drift state (`.speckit/`) | **yes** | content hashes; offline-verifiable; engine I/O |
| Code, tests, `// SPEC:` pointers | **yes** | the implementation under proof |
| Agent session memory / project notes | **yes** (markdown) | durable context belongs with the code, not a pinned issue |
| Defects | no → **Issues** | ephemeral intake; its durable form is a regression scenario in the repo |
| Work / epics / the agent's task board | no → **Projects** | ephemeral coordination |
| Gating | no → **PRs + Actions** | enforcement trigger, not a proof input |

Hard rule: **nothing the engine must verify or hash leaves the repo.** Everything on
GitHub is disposable — you could delete the whole board and lose no truth.

## Shape: `specify` *is* a `gh` extension

One Go binary, two invocation paths:

- **standalone `specify`** — installed via Homebrew / Mise / release binary. The
  engine commands (`scan`/`verify`/`lock`/`drift`/`cover`/`parity`/`gate`) need no
  `gh` and no network. Offline integrity is preserved.
- **`gh specify …`** — installed via `gh extension install markmals/specify` (a
  `gh` extension is just a binary named `gh-specify` that `gh` dispatches to). This
  is the GitHub-native install, and the reason it's worth it: the binary
  **inherits `gh`'s auth** (`gh auth token`), so every GitHub call — REST or
  GraphQL — needs zero separate token plumbing.

Consequences for the design:

- **We inline the Projects GraphQL ourselves.** Rather than depend on a separate
  `gh` extension, the binary contains the GraphQL client for everything `gh project`
  can't do (item/field CRUD beyond status, sub-issue progress, dependency edges).
  We lift the approach from `NSExceptional/gh-projects` and **vendor it directly in
  this repo** (open decision 4), evolving it internally.
- **GitHub commands live in the binary, un-namespaced.** No mandatory `github`
  subcommand wall — `specify` calls GitHub APIs/services directly where it helps
  (open decision 3). It uses `gh`'s token for auth and shells out to `gh` for the
  blessed commands `gh` already does well.
- **Config shrinks toward zero.** Because the extension inherits `gh`'s repo context
  (current repo, the linked project) and auth, we likely **don't need a `github`
  config block at all** (open decision 1). The only thing not inferable is *which
  project board* — and `gh` can list a repo's linked projects, so even that may be
  auto-detected, with a flag/one-line config as the fallback.

## Pillar 1 — PR gating (build first)

The engine becomes a **required status check**: spec-honesty is non-bypassable.
`specify gate firewall` as a PR check means *you cannot merge a test that silently
drifted from its spec* — that's the demo.

### Custom Action **and** inline — layered

An official composite action (`markmals/speckit/gate@v1`) that:

1. installs the pinned `specify` binary,
2. runs the gate for the target: `scan` → `verify <target>` → `parity <target> --gate` → `gate firewall` / `gate generated` / `gate scope`,
3. emits **Checks-API annotations** mapping each unjoinable scenario, dangling test
   reference, or drifted spec to its exact file + line.

### Two gates, one workflow, two jobs

A PR run has two distinct concerns, and they get **two parallel jobs in one
`ci.yml`** — not one crammed job, and not two separate workflow files (same
trigger → one run to inspect, but independent status checks):

- **`quality`** — the target's fast static checks via its mise tasks:
  `fmt:check`, `lint`, `typecheck`.
- **`verify`** — the SpecKit spec gate: `scan` → `verify <target>` → `parity --gate`
  → `gate firewall`/`generated`/`scope`, with Checks-API annotations.

Crucially, **`specify verify` already runs the target's test suite** (that's how the
join works), so tests live in the `verify` job — they are *not* run twice. The
`quality` job is only the static trio. The two jobs run in parallel (a type error
and a spec drift surface independently and at once), and both are required checks.

**Workflow files are named for what they do** (not `speckit.yml`). The per-stack
file is a thin caller; the `verify` job delegates to the official reusable workflow
(so SpecKit owns/updates it via the tag), while `quality` runs the stack's own
tasks:

```yaml
# .github/workflows/ci.yml   (dropped in by `target add`)
on: { pull_request: {} }
concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: true }
jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: jdx/mise-action@v3
      - run: mise run -C apps/web fmt:check
      - run: mise run -C apps/web lint
      - run: mise run -C apps/web typecheck
  verify:
    uses: markmals/speckit/.github/workflows/gate.yml@v1
    with: { target: web }
```

Pure inline `run: specify verify` for the spec gate loses the **annotations** (the
killer feature — inline PR comments on the exact scenario), duplicates logic across
stacks, and forces a re-scaffold to update. The reusable workflow earns its keep via
annotations + setup + single-source updates; the thin caller keeps it transparent.
This works stack-agnostically because every scaffold exposes the **standard mise
task names** (`test`, `fmt:check`, `lint`, `typecheck`); a stack without one simply
omits that step. (Mise task-name parts are colon-separated — `fmt:check`, not
`fmt-check`.)

### Branch protection / rulesets

The gate only bites if **required**. Rulesets are repo settings, not a checked-in
file, so `specify` provisions them via the GitHub API (it has `gh`'s token):
require the SpecKit checks, require a PR, restrict force-pushes. Ship a documented
fallback recipe in the scaffold README.

## Pillar 2 — Issues as ephemeral defect intake

### Lifecycle (scenario-canonical, no durable link)

```
defect filed (Issue, type Bug, via defect.yml form, auto-added to the board)
  → triage
  → the fix updates/adds a SCENARIO in the repo (if spec-representable)
      OR adds a regression TEST bound to a new scenario (if it can't/shouldn't live in the spec)
  → fix PR; merge → verify <target> joins the new scenario green → writes the lock
  → close the issue (the lock is the proof; the issue was just intake)
```

The **Issue is ephemeral intake + history; the scenario + its lock are the truth.**
We deliberately keep **no durable issue↔scenario backlink** (open decision 2) — the
issue can be closed, archived, even deleted, and nothing in the repo depends on it.
GitHub's automatic cross-references (the fix PR mentioning `#123`) are enough of a
breadcrumb; we don't maintain a bidirectional link.

### Issue types & epics

Use org **Issue Types** (GA 2025-04): Bug / Feature / Task plus a custom **Epic**
type. Epics are an Epic-typed issue with **sub-issues** (GA: 100/parent, 8 deep) —
the same hierarchy Beads gets from parent-child. *Caveat:* Issue Types are an
**organization** feature; a personal repo falls back to a `type:` label.

### What goes in each target's `.github/`

`target add` drops in (descriptive filenames throughout):

| File | Purpose |
| --- | --- |
| `workflows/ci.yml` | PR checks — a `quality` job (fmt:check/lint/typecheck) + a `verify` job (the spec gate, which also runs tests) |
| `workflows/deploy.yml` | *optional* deploy (see below); only if a deploy target was chosen |
| `PULL_REQUEST_TEMPLATE.md` | spec-touch checklist (specs changed? scenarios bound? `verify` green? `drift` clean?) |
| `ISSUE_TEMPLATE/defect.yml` | a defect form — stamps `type: Bug`, a label, the project; prompts for repro + the target |
| `ISSUE_TEMPLATE/config.yml` | points docs at *this* repo |
| `CODEOWNERS` | maps `/features/**` and `/specs/**` to the spec owner, so **spec changes require human review** |
| `dependabot.yml` | for the stack's ecosystem (web → npm, go → gomod) |

## Pillar 3 — Projects as the work surface (Beads-informed, simplified)

The board is **ephemeral coordination**, projected from work state and driven by the
agent via the inlined GraphQL client. It is a kanban: **the "ready queue" is just a
Status column** (e.g. Backlog → Todo → In Progress → In Review → Done), not a
computed field. The agent pulls the top card of the actionable column and moves it
across as it works.

> Column set TBD — mirroring Mark's `APL-Innovation-Lab/projects/1` once its columns
> are confirmed (couldn't read it: the token lacks `read:project`).

### What we borrow from Beads — and what we drop

Beads (`bd`, Steve Yegge) is structured, queryable agent memory: a dependency-aware
issue graph that replaces bit-rotting markdown plans. We steal the *patterns* that
survive a simpler, column-based model:

| Beads pattern | Our decision |
| --- | --- |
| **parent-child** (epic → subtask) | **keep** → Epic issue type + sub-issues (native) |
| **`ready` = open ∧ unblocked**, computed | **drop the computation** → "ready" is a kanban column you move cards into |
| **`blocks` / `blocked-by`** | **keep as a signal only** → native dependency badge / `is:blocked` filter so the agent doesn't pull a blocked card; not an automated gate |
| **`discovered-from` provenance** | **keep as convention** → file mid-task follow-ups as new issues with a `discovered-from:#N` label + body backlink (GitHub has no native provenance edge — the one gap) |
| **atomic claim** | **keep** → assign-self + move to In Progress in one mutation (GitHub assignment is server-atomic; multi-agent safe) |
| **collision-free hash IDs** | **drop** → GitHub allocates `#N` atomically server-side; not needed |
| **`bd remember` / `bd prime`** (pinned memory) | **drop from GitHub** → durable agent memory + session context stay as **repo markdown** |
| **"land the plane" teardown** | **keep** → an end-of-session ritual (a skill): run the gate, file discovered work, push, hand off |

### The agent loop (on GitHub, via `specify` + `gh`)

```
prime    → read session context from REPO markdown (memory/, AGENTS.md) — not GitHub
ready    → take the top card of the actionable column (skip any showing "blocked")
claim    → assign self + move to In Progress  (one atomic mutation)
work     → implement tests-first; bind each test to its scenario
discover → file follow-ups mid-task as issues, labeled discovered-from:#N
verify   → specify verify <target> → join → lock
close    → move to Done / close the issue ON GREEN (lock = proof)
land     → run the gate, confirm discovered work is filed, push, post a handoff
```

### Operational reality (from the research)

- **Projects is GraphQL-only** (no REST). The inlined client is a two-tier thing:
  the `gh project` verbs for status/field CRUD, raw GraphQL for sub-issue progress,
  dependency edges, and automations.
- **Cache node IDs** — the API takes IDs, not names; resolve project/field/option
  IDs once.
- **Idempotency:** adding an item that's already present returns the existing item;
  field writes are last-write-wins.
- **Rate budget:** GraphQL shares 5,000 points/hr — use `gh`'s token; never let a
  board-sync failure block a local `verify`.
- **Built-in automations** (close → Done, PR-merged → Done, linked-PR → In progress)
  are free but **UI-configured only** — document the setup; can't be provisioned via
  `gh`. **Multi-select fields are not available yet.**

## Deploy workflows (optional, configurable)

Optional GitHub deploy workflows, **none required**, chosen at `specify init` (a
`--deploy <kind>` flag / prompt) and **addable after the fact** (`specify deploy add
<kind>`; exact ergonomics — project-level vs per-target — is open). Each drops a
descriptively-named `.github/workflows/deploy.yml`, triggered on push to the default
branch (or on release), and lists the secrets to set (via `gh secret set` or the UI).

| kind | Action / mechanism | Secrets | Notes |
| --- | --- | --- | --- |
| `cloudflare-workers-ssr` | `cloudflare/wrangler-action` → `wrangler deploy` | `CLOUDFLARE_API_TOKEN` | SSR app on Workers (e.g. TanStack Start); `account_id` is committed in `wrangler.jsonc` (an identifier, not a secret), which also carries the server entry |
| `cloudflare-workers-spa` | `wrangler deploy` with Workers static **assets** | `CLOUDFLARE_API_TOKEN` | assets-only; `account_id` in `wrangler.jsonc`; `assets.not_found_handling = "single-page-application"`, no server worker |
| `railway` | Railway CLI in-workflow (`railway up --service <svc>`) | `RAILWAY_TOKEN` | for server/container apps; alternatively Railway's own GitHub auto-deploy (no workflow) |
| `github-pages-spa` | `actions/upload-pages-artifact` + `actions/deploy-pages` | none (uses `GITHUB_TOKEN`) | needs Pages enabled, `pages: write` + `id-token: write`, `environment: github-pages`; handle the `/<repo>/` base + SPA 404 fallback |

These are independent of the gate (deploy on push/release; gate on PR).

## Secrets — 1Password (`op`) as the source of truth

Same discipline as specs: **one source of truth, projected outward.** Secrets live
in **1Password**; SpecKit wires them into the places that need them via the `op`
CLI. Values are **never committed and never printed** — the repo holds only `op://`
*references* (pointers like `op://Private/Cloudflare/api_token`), which are safe to
commit: they name a vault/item/field, not a value (the same way the project already
stores the Homebrew tap PAT as a 1Password reference).

Two destinations, both sourced from `op`:

1. **GitHub Actions secrets** — the CI deploy credentials:
   `gh secret set CLOUDFLARE_API_TOKEN --body "$(op read op://Private/Cloudflare/api_token)"`.
2. **The deploy platform's own secret store** — the app's *runtime* secrets:
   - Cloudflare: `op read op://… | wrangler secret put NAME`
   - Railway: `railway variables --set "NAME=$(op read op://…)"`

Declarative — the deploy manifest maps each secret to its reference (committable):

```jsonc
"deploy": {
  "kind": "cloudflare-workers-ssr",
  "ci":      { "CLOUDFLARE_API_TOKEN": "op://Private/Cloudflare/api_token" },
  "runtime": { "DATABASE_URL": "op://Private/app-db/url" }
}
// note: CLOUDFLARE_ACCOUNT_ID is not here — it's committed in wrangler.jsonc (an identifier, not a secret)
```

`specify deploy add <kind>` (and a re-runnable `specify secrets sync`) reads each
reference through the developer's locally-authenticated `op` and pushes it to GitHub
/ the platform. Values are **piped straight from `op` into `gh` / `wrangler` /
`railway`** — never echoed, logged, or written to disk.

**Optional upgrade — runtime load (no copies anywhere).** For CI deploy credentials,
instead of `gh secret set`-ing copies, store a single `OP_SERVICE_ACCOUNT_TOKEN` in
GitHub and have `deploy.yml` resolve the `op://` references at deploy time via
`1password/load-secrets-action`. Then 1Password is the *only* place the secret
exists and rotation is one edit. Recommended once a deploy is real; the `gh secret
set` sync is the simpler default (open decision 7).

`op` is pinned alongside `gh` in `mise.toml`. Local use relies on the 1Password
desktop-app integration; CI runtime-load uses a service-account token.

## GitHub features we lean on, and version pins

| Feature | Status (2026-06) | We use it for |
| --- | --- | --- |
| `gh` `--type` / `--parent` / `--blocked-by` + JSON | GA in **gh ≥ 2.94.0** (2026-06-10) | the automation surface — **pin `gh ≥ 2.94.0` via `mise.toml` (`[tools] gh = "2.94"`)** |
| Issue types (org) incl. custom Epic | GA 2025-04 (**org-only**) | defect = Bug; epics = Epic + sub-issues; label fallback off-org |
| Sub-issues (100/parent, 8 deep) | GA 2025-04 | epic → sub-issue hierarchy |
| Issue dependencies (`blocked-by`) | GA 2025-08 | a "blocked" badge so the agent skips blocked cards (signal, not gate) |
| Issue forms (auto type/label/project) | GA | the `defect.yml` intake |
| Projects GraphQL item/field CRUD; built-in automations | GA | board sync + free status transitions |

Pin `gh` **and `op`** (the 1Password CLI) as peer dependencies in **`mise.toml`**
(repo and scaffolds), so both are present wherever `specify` runs. Preview — **don't depend on yet:** org-wide
Issue Fields (preview); multi-select Project fields (not shipped).

## Future Directions

- **VS Code extension** — codelens on `// SPEC:` pointers (jump scenario ↔ bound
  test), drift indicators in the gutter, a parity tree of the spec library, run the
  gate from the editor, a board view. The developer-native complement to the
  agent-native CLI.
- **Discussions** — *maybe* — spec RFCs (proposed specs debated before they're
  committed). Take it or leave it; not a priority.

(Explicitly **not pursuing:** the GitHub MCP `projects` toolset and GitHub Agentic
Workflows — the chosen path is agent-native CLIs + skills that teach agents to drive
them, not MCP or GitHub-hosted orchestration.)

## Open decisions

1. **Config block** — start with a tiny optional `github` block *only* for what `gh`
   can't infer (the project number); likely droppable once the extension auto-detects
   the repo's linked project. Lean on `gh` context + auth.
2. **No issue↔scenario link** — issues/projects are ephemeral; rely on GitHub's
   automatic cross-references, maintain nothing. (Decided.)
3. **GitHub commands in the binary, un-namespaced** — `specify` calls GitHub
   APIs/services directly; no mandatory `github` subcommand wall. (Decided.)
4. **Inline the Projects GraphQL** (lifted from `gh-projects`) **directly in this
   repo** and evolve it internally; no upstream dependency. (Decided.)
5. **Ready is a board column, not a computed field.** (Decided.)
6. **Deploy command ergonomics** — `init --deploy` + `specify deploy add`; is deploy
   project-level or per-target/app? Lean per-target (an app deploys), with `init`
   sugar for the primary app. (Open.)
7. **Default secret mode** — `gh secret set` sync (simpler; copies into GitHub) vs
   runtime-load via `OP_SERVICE_ACCOUNT_TOKEN` + `1password/load-secrets-action` (no
   copies; one-place rotation). Lean: sync as the default, runtime-load as a
   documented upgrade. (Open.)
