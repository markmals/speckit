# Design — GitHub-native integration

**Status:** proposal / direction-setting. Captures the pivot from "Claude-native"
(Workbench) to **GitHub-native**: SpecKit stays agent-native and agent-agnostic
(Claude Code / Codex / Copilot today), but its *banner* becomes that the framework
*lives on GitHub* — PRs gate, Issues hold defects, Projects hold the agent's work —
instead of being kept as repo markdown. Research current as of 2026-06.

## Thesis

> The repo holds the **truth**; GitHub holds the **process**; the engine projects one onto the other.

Two layers, joined by `specify`:

- A **portable spec-integrity core** — git + files + test runners; the engine
  (`scan`/`verify`/`lock`/`drift`/`cover`/`parity`/`gate`); works offline; never
  reads the GitHub API for *correctness*. This is the guarantee.
- A **GitHub-native workflow shell** — Issues / Projects / PRs / Actions. This is
  the banner.

The mechanic that makes it *SpecKit* and not just "please use GitHub" is the same
one we already use for code: **projection.** Code is a materialized view of specs;
now GitHub surfaces are materialized views of *engine state*. `specify` projects
truth outward (opens issues for unbound scenarios, syncs project items from
drift/parity, sets PR check status) and the agent reads coordination back via
`gh`. *Everything the engine reports must be earned* now extends to the board and
the PR checks — they can't lie, because the engine owns them.

## The determinism line (what may move, what may not)

The test for every artifact: **does the engine have to verify, hash, or diff this
at a specific commit?**

| Artifact | Stays in repo? | Why |
| --- | --- | --- |
| Scenarios / acceptance criteria / models | **yes** | the join hashes & diffs them per commit; the BDD source |
| Locks, parity, drift state (`.speckit/`) | **yes** | content hashes; offline-verifiable; engine I/O |
| Code, tests, `// SPEC:` pointers | **yes** | the implementation under proof |
| Defects | no → **Issues** | mutable work item; its durable form is a regression scenario in the repo |
| Work / the agent's plan & tasks | no → **Projects** | coordination; already treated as disposable |
| Gating | no → **PRs + Actions** | enforcement trigger, not a proof input |

Hard rule: **nothing the engine must verify or hash leaves the repo.** GitHub
carries process *around* the specs, never the specs themselves.

## Blessed-default-but-overridable (the provider seam)

GitHub is a **hard requirement of the blessed path** — and that's deliberate. On
that path we go deep: official Action, `specify` forwards to the `gh` CLI for
blessed commands, the engine reads Issues/Projects directly. We don't pretend
everything is abstract.

But the integration is **overridable**, because the integrity core doesn't depend
on it:

- Run `specify` manually in a GitLab CI pipeline instead of the official Action.
- Point your agent at `linctl` + Linear for issue/work tracking instead of `gh`.

The seam is intentionally **light** for v1 — config selects providers; the engine
core is provider-free. Default shape in `.speckit/specs.json`:

```jsonc
{
  "github": {                  // presence enables the blessed path; omit to opt out
    "repo": "markmals/spec-kit",   // optional — gh infers from origin
    "project": 4,                  // the Projects board number
    "gate": ["scan", "verify", "parity", "firewall", "generated"]
  }
}
```

Overriding = omit the `github` block (the engine still fully works) and wire your
own CI/agent skills. A fuller `"providers": { "ci": …, "issues": …, "work": … }`
map is a future option if real demand for a second blessed provider appears — we
won't build the abstraction before we need it.

---

## Pillar 1 — PR gating (build first)

The strongest, cheapest banner: the engine becomes a **required status check**, so
spec-honesty is non-bypassable. `specify gate firewall` as a PR check means *you
cannot merge a test that silently drifted from its spec* — that's the demo.

### Custom Action **and** inline — layered (decision)

Ship an **official composite action** (e.g. `markmals/speckit/gate@v1`) that:

1. installs the pinned `specify` binary (setup-style),
2. runs the gate for the target: `scan` → `verify <target>` → `parity <target> --gate` → `gate firewall` / `gate generated` / `gate scope`,
3. emits **Checks API annotations** mapping each unjoinable scenario, dangling
   test reference, or drifted spec to its exact file + line.

The scaffolded per-stack workflow is then a **thin caller**:

```yaml
# .github/workflows/speckit.yml (dropped in by `target add`)
on: { pull_request: {} }
jobs:
  gate:
    uses: markmals/speckit/.github/workflows/gate.yml@v1   # reusable workflow
    with: { target: web }
```

So it *reads* per-stack but *delegates* to the versioned action. Why not pure
inline `run: specify verify`?

- **Annotations** — the killer feature. Inline `run:` can surface a red X; only an
  action that speaks the Checks API can put *"scenario.welcome.greet.hello has no
  bound test"* on the exact line of the PR diff. That's the thing that makes the
  gate feel native.
- **Central updates** — fix the gate once; every consumer repo gets it via the tag.
  Inline means re-scaffolding every repo.
- **Marketplace presence** — an official "SpecKit" Action is itself the
  GitHub-native banner.

The per-stack thinness keeps it transparent and customizable; the action earns its
keep via annotations + setup + single-source updates.

### Branch protection / rulesets

The gate only bites if it's **required**. Rulesets aren't a checked-in file
(they're repo/org settings), so we provision them via `gh api` — a future
`specify github protect` subcommand that requires the SpecKit checks, requires a PR
before merge, and restricts force-pushes. Until then, ship a documented `gh`
recipe in the scaffold's README.

---

## Pillar 2 — Issues as the defect intake

### Lifecycle (scenario-canonical)

```
defect filed (Issue, auto-typed Bug, auto-added to board)
  → triage
  → the fix updates/adds a SCENARIO in the repo (if the behavior is spec-representable)
      OR adds a regression TEST bound to a new scenario (if it can't/shouldn't live in the spec)
  → fix PR references the issue WITHOUT auto-closing it
  → merge → `verify <target>` joins the new scenario green → writes the lock
  → THEN close the issue (the lock is the proof the defect is fixed)
```

The **Issue is intake + history; the scenario + its lock are the truth.** A defect
never lives only as a closed issue — it lives as a scenario you can re-verify
forever.

This is enabled by a real, current GitHub feature: the **link-without-auto-close
repo setting** (GA 2025-04-23). The fix PR can say `Fixes #123` for the link graph
while the issue stays open until the scenario actually verifies green — closing on
*proof*, not on *merge*.

### Linking convention

The scenario carries the originating issue, and the lock records it, so
issue ↔ scenario ↔ test ↔ lock is a closed loop:

```md
## Scenario: rejects an empty name   <!-- id: scenario.welcome.greet.empty -->
<!-- issue: markmals/spec-kit#123 -->
```

### Issue types, not labels

Use **org Issue Types** (GA 2025-04, Bug/Task/Feature, customizable) rather than
`type:` labels — a defect is `type: Bug`. They're a real field, filterable, and an
issue form can stamp the type at intake.

### What goes in each target's `.github/`

`target add` should drop in:

| File | Purpose |
| --- | --- |
| `workflows/speckit.yml` | the gate (thin caller of the official reusable workflow) |
| `PULL_REQUEST_TEMPLATE.md` | spec-touch checklist (specs changed? scenarios bound? `verify` green? `drift` clean?) |
| `ISSUE_TEMPLATE/defect.yml` | a defect form — stamps `type: Bug`, a label, the project, and prompts for repro + the target + the scenario id it violates |
| `ISSUE_TEMPLATE/config.yml` | points discussions/docs at *this* repo |
| `CODEOWNERS` | maps `/features/**` and `/specs/**` to the spec owner, so **spec changes require human review** — reinforces "deviations need sign-off" |
| `dependabot.yml` | for the stack's ecosystem (web → npm, go → gomod) |

Branch protection / the ruleset is repo settings (provisioned via
`specify github protect` / `gh`), not a file.

---

## Pillar 3 — Projects as the work surface (Beads-informed)

The board is a **materialized view of engine + work state.** `specify` projects
open issues for unbound scenarios and drifted specs, creates/updates project items,
and sets a `spec-id` text field, `Status`, and `Priority`. The agent reads "what's
next" from the board and drives it via `gh`.

### What we borrow from Beads (repointed at GitHub)

Beads (`bd`, Steve Yegge) is structured, queryable, git-synced *memory for agents*
— a dependency-aware issue graph that replaces bit-rotting markdown plans. We don't
adopt Beads's storage; we steal its **patterns**, implemented on GitHub primitives.
The crux of Beads is that **only `blocks` + parent→child propagation gate
readiness**; `discovered-from` and `related` are pure metadata.

| Beads pattern | GitHub implementation | Fit |
| --- | --- | --- |
| **parent-child** (epic → subtask) | **Sub-issues** (GA, 100/parent, 8 deep) | clean |
| **blocks / blocked-by** | **Issue dependencies** (GA Aug 2025, `is:blocked` filter, `gh --blocked-by`) | clean |
| **`ready` queue** (open ∧ no open blocker ∧ no blocked ancestor) | a **computed `Ready` boolean Project field** + a "Ready" view sorted by Priority | needs scripting — GitHub won't compute *transitive* unblock for you |
| **`discovered-from`** provenance | a `discovered-from:#N` label + body backlink + GitHub's auto cross-reference | **the one gap** — a convention, not a typed queryable edge |
| **atomic `--claim`** | assign-self + `Status: In progress` in one GraphQL mutation | clean — GitHub assignment is server-atomic, so multi-agent claims are safe |
| **collision-free hash IDs** | not needed — GitHub allocates `#N` atomically server-side | GitHub is *simpler* here |
| **`bd remember` / `bd prime`** | a pinned "Project Memory" issue + the Ready view, fetched at session start | clean (complements SpecKit's file memory) |
| **"land the plane" teardown** | an end-of-session ritual (skill + optional close-out check) | clean |

The **`Ready` field is the highest-leverage borrow.** GitHub gives you the
dependency edges and the `is:blocked` filter, but not transitive readiness across a
parent chain — so a small `gh`/GraphQL step (on-demand via `specify`, or a
scheduled Action) sets `Ready = true/false` per open issue, and the agent's queue
is just the "Ready" view sorted by Priority. That reproduces `bd ready --json`;
emit the blocker list as the `--explain` reason.

### The agent loop (Beads's loop on GitHub, via `specify` + `gh`)

```
prime    → read the Ready view + the pinned Project-Memory issue (session context)
ready    → pick the top unblocked item (gh projects ready / the Ready view)
claim    → assign self + Status: In progress  (one atomic mutation)
work     → implement tests-first; bind each test to its scenario
discover → file follow-ups mid-task as issues, linked discovered-from:#N
verify   → specify verify <target> → join → lock
close    → close the issue ON GREEN (lock = proof); recompute Ready for unblocked items
land     → run the gate, confirm discovered work is filed, push, post a handoff
```

### The `gh-projects` extension

Use **`NSExceptional/gh-projects`** as the agent's Projects CLI — it wraps the
Projects v2 GraphQL API ergonomically (address projects by number, items by issue
number/title, fields/options by *name*), and already ships the verbs we need:
`create`, `add`, `draft`, `move`, `set` (incl. iteration + text fields), `ready`
(= items in Todo), `field-create`, `link`. **18 subcommands; it even has a `ready`.**

Caveats (it's a day-old v0.1.x, single author): **fork and pin** rather than depend
on upstream. Two cheap PRs close the only real gaps, worth sending back:

1. `--json` on the write commands (so the agent can capture the created item id).
2. a generic field-value query (`items --field X --value Y`) — today only `Status`
   is a first-class filter.

Fallback for both: raw `gh api graphql` (the extension's own `internal/projects/`
shows how). Don't gate our design on upstream acceptance — vendor the fork.

### Operational reality (from the research)

- **Projects is GraphQL-only** (no REST). Plan a two-tier client: `gh project` /
  `gh-projects` for item+field CRUD, raw GraphQL for automations, sub-issue
  progress, and dependency edges.
- **Cache the IDs.** The API takes node IDs, not names — resolve and cache project
  id, field ids, and single-select **option ids** once (`field-list`).
- **Idempotency:** `addProjectV2ItemById` returns the existing item if already
  present (safe to re-run); field writes are last-write-wins.
- **Rate budget:** GraphQL shares 5,000 points/hr — use a GitHub App token for
  headroom; never let a board-sync failure block a local `verify`.
- **Multi-select fields are NOT available yet** — don't design fields around them.
- **Built-in automations** (close → Done, PR-merged → Done, linked-PR → In
  progress) are free but **UI-configured only** — document the setup; you can't
  provision them via `gh`.

---

## GitHub features we lean on (and version pins)

| Feature | Status (2026-06) | We use it for |
| --- | --- | --- |
| `gh` `--type` / `--parent` / `--blocked-by` + JSON | GA in **gh ≥ 2.94.0** (2026-06-10) | the whole automation surface — **pin `gh ≥ 2.94.0`** |
| Issue types (org) | GA 2025-04 | defect = `type: Bug` |
| Issue dependencies | GA 2025-08 | `blocks`/`blocked-by` for the Ready queue |
| Sub-issues (100/parent, 8 deep) | GA 2025-04 | spec/epic → scenario hierarchy |
| Link-without-auto-close toggle | GA 2025-04 | close defects on *proof*, not on merge |
| Issue forms (auto type/label/project) | GA | the `defect.yml` intake |
| Projects built-in automations | GA | free status transitions (UI-configured) |
| Projects GraphQL item/field CRUD | GA | engine → board sync |
| Parent / Sub-issue-progress Project fields | GA | rollup without computing it |

Preview — **do not depend on yet:** org-wide Issue Fields (preview, May 2026);
multi-select Project fields (not shipped). Semantic/hybrid issue search (GA
2026-04) is a nice-to-have for "find the defect/scenario like this."

---

## Future Directions

- **VS Code extension** — the developer-native complement: codelens on `// SPEC:`
  pointers (jump scenario ↔ bound test), drift indicators in the gutter, a parity
  tree view of the spec library, run the gate from the editor, and a board view of
  the Ready queue. Agent-native *and* developer-native.
- **`specify` as a `gh` extension** — `gh speckit verify web`, installed via
  `gh extension install`. A genuinely GitHub-native install story alongside the
  Homebrew/Mise paths.
- **GitHub MCP server `projects` toolset** (GA 2025-10) — an alternative to
  `gh-projects` for MCP-native agents; the engine could speak MCP instead of
  shelling out to `gh`.
- **GitHub Agentic Workflows** (public preview 2026-06) — GitHub-hosted agent
  orchestration; could host the SpecKit loop server-side rather than in the
  developer's agent.
- **Discussions** — spec RFCs (proposed specs debated before they're committed) and
  methodology Q&A. Weakest fit; only if the community wants it.
- **Packages / Releases** — distribute platform packs and stack scaffolds as
  versioned artifacts, not just the binary.
- **Org-wide Issue Fields** (when GA) — a standard Priority/Effort across every repo
  in an org, instead of per-project fields.

---

## Open decisions

1. **Provider config shape** — the light `"github"` block (v1) vs a full
   `"providers"` map. Recommend the block until a second blessed provider is real.
2. **Where the issue↔scenario link lives** — scenario frontmatter, the lock, or
   both. Recommend scenario comment + lock record (the loop above).
3. **`specify github …` subcommands vs pure skills** — do we build
   `github protect` / `github sync` / `github issue→scenario` into the binary, or
   keep GitHub orchestration in agent skills that drive `gh`? Likely a thin set of
   binary subcommands for the deterministic bits (protect, the Ready computation,
   the projection sync) + skills for the judgment bits.
4. **`gh-projects` fork ownership** — vendor under `markmals/` and pin; attempt the
   two upstream PRs but don't depend on them.
5. **Ready-field computation** — on-demand via `specify` vs a scheduled Action.
   Probably both: `specify` computes on sync; an Action keeps it fresh.
