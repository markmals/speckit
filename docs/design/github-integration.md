# Design — GitHub integration

**Status:** as built, trimmed to what shipped and survived the stack-agnostic
refactor. SpecKit stays agent-native and agent-agnostic; GitHub is an optional
workflow shell around a repo-local engine. The commands this doc once specified
for deploys, secrets, branch-protection provisioning, and a GitHub-only
issues/work surface were removed — work tracking is now a pluggable provider
([../work-providers.md](../work-providers.md)), and deploys/secrets/rulesets
belong to the adopting project's own tooling.

## Thesis

> The repo holds the **truth**; GitHub holds the **process**; the engine projects one onto the other.

Two layers, joined by `specify`:

- A **portable spec-integrity core** — git + files + test reports; the engine
  (`scan`/`verify`/`lock`/`drift`/`cover`/`parity`/`gate`); works offline; never
  needs GitHub for *correctness*. This is the guarantee.
- An **optional GitHub workflow shell** — PR gating via Actions, and the
  `github-projects` work provider.

The durable artifacts (specs, scenarios, locks, tests) live in the repo; work
items and PR checks are throwaway coordination around the work.

## The determinism line (what may move, what may not)

The test: **does the engine have to verify, hash, or diff this at a specific commit?**

| Artifact | Stays in repo? | Why |
| --- | --- | --- |
| Scenarios / acceptance criteria / models | **yes** | the join hashes & diffs them per commit; the BDD source |
| Locks, parity, drift state (`.speckit/`) | **yes** | content hashes; offline-verifiable; engine I/O |
| Code, tests, `// SPEC:` pointers | **yes** | the implementation under proof |
| Agent session memory / project notes | **yes** (markdown) | durable context belongs with the code, not a pinned issue |
| Work items / defect intake | provider's choice — a committed `WORK.md` by default; a Projects board with the `github-projects` provider | ephemeral coordination; its durable form is a regression scenario in the repo |
| Gating | no → **PRs + Actions** | enforcement trigger, not a proof input |

Hard rule: **nothing the engine must verify or hash leaves the repo.** The
engine packages (`internal/engine`, `internal/specmodel`, `internal/reports`,
`internal/config`) import neither the GitHub client nor any work provider —
providers are wired only in `cmd/specify` — so a network failure can never
block a local `verify`. That isolation is structural, not conventional.

## Auth: inherit `gh`, plumb nothing

The GitHub client (`internal/github`) inherits `gh`'s token (`gh auth token`,
env fallback) and auto-detects the repo via `gh repo view`, so there is **no
token config and no `github` block** in `.speckit/specs.json`. The
`github-projects` work provider carries only what can't be inferred: the board
number and (optionally) its owner. Projects v2 is GraphQL-only, so the client
inlines its own GraphQL queries — no external `gh` extension dependency.

The same binary also works as a `gh` extension (`gh specify …` — a `gh`
extension is just a binary named `gh-specify` that `gh` dispatches to); the
standalone install path never needs `gh` at all.

## PR gating (built)

The engine as a **required status check**: `specify gate firewall` as a PR check
means *you cannot merge a test that silently drifted from its spec*. The
composite action (`markmals/speckit/gate@v0.2.0`) and the reusable workflow
(`gate.yml`) run `scan` → firewall → `verify <target>` → `parity --gate`, each
with `--format github` so failures annotate the exact `file:line` in the PR.
The gate is stack-neutral: it runs the target's configured `command` and the
specify checks, nothing else — toolchain setup belongs to the calling workflow.
Mechanics, the CI shapes, and the branch-protection `gh` recipe:
[../ci-gating.md](../ci-gating.md).

## Work tracking (moved)

The Beads-informed board design (ready-as-a-column, atomic claim, canonical
states) survives as the **`github-projects` provider** behind the
provider-agnostic `specify work` surface. The comparison of providers — the
committed-markdown default, Beads' typed dependencies and computed readiness,
the Projects board, and `none` — lives in
[../work-providers.md](../work-providers.md).

## Future directions

- **VS Code extension** — codelens on `// SPEC:` pointers (jump scenario ↔ bound
  test), drift indicators in the gutter, a parity tree of the spec library, run
  the gate from the editor. The developer-native complement to the agent-native
  CLI.

(Explicitly **not pursuing:** the GitHub MCP `projects` toolset and GitHub
Agentic Workflows — the chosen path is agent-native CLIs + skills that teach
agents to drive them, not MCP or GitHub-hosted orchestration.)
