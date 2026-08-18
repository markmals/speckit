# Work providers — `specify work`

`specify work` is SpecKit's work-tracking surface: a small set of verbs over a
pluggable **provider**. The provider is workflow plumbing, not proof — the
engine never touches it.

## The hard invariant

`scan`, `verify`, `drift`, `cover`, `parity`, and `gate` never require, invoke,
or import a work provider. Providers are wired only in the CLI's `work`
subcommands, and an absent `work` block in `.speckit/specs.json` is never an
error — it just means the default (`markdown`) provider. Deleting your entire
work state loses nothing the engine verifies.

## The verbs

Every verb takes `--json` for machine-readable output.

| Command | What it does |
| --- | --- |
| `specify work ready` | List the actionable items — everything in the `ready` state. |
| `specify work create <title> [--type task\|defect] [--spec <spec-id>]` | File a work item; it lands in `ready`. `--spec` records the spec ID the item advances. |
| `specify work claim <id>` | Take an item: it moves to `in-progress`. |
| `specify work move <id> <state>` | Move an item to a state. |
| `specify work list [--state <state>]` | List items, optionally filtered by state. |

## States and types

The canonical states, in lifecycle order:

```text
ready → in-progress → blocked → done
```

`create` lands an item in `ready`; `claim` moves it to `in-progress`. Every
provider maps these four onto its own vocabulary (the mappings are listed per
provider below).

Two item types: **`task`** (the default) and **`defect`**. Defect intake is
`specify work create "<title>" --type defect` — a defect's durable form is
still a regression scenario plus a bound test in the repo; the work item is
ephemeral coordination around getting there.

## Choosing a provider

The `work` block in `.speckit/specs.json` selects one of four providers:
`markdown` (the default), `beads`, `github-projects`, or `none`.

### `markdown` (default) — a committed file

No network, no external binary, no dependency graph. Work state is one
committed markdown file (default `WORK.md`), precise enough to hand-edit and
diff:

```json
"work": { "provider": "markdown", "file": "WORK.md" }
```

(`file` is optional — it defaults to `WORK.md`. An absent `work` block means
exactly this provider.)

**The file format.** Sections *are* the states: a `## <Heading>` opens a state
whose name is the heading slugified (`## In progress` → `in-progress`). The
canonical four states are always rendered, in canonical order, even when empty;
any other heading is a valid extra state (kept in first-seen order, dropped when
empty). Text before the first heading is preamble and survives verbatim.

Each item is one line: a task-list checkbox (`- [ ] `, or `- [x] ` for items in
`done`), the id in backticks, the title, then optional ` · `-separated fields
`spec: <spec-id>` and `type: defect`:

```markdown
# Work

## Ready

- [ ] `wk-3` Wire the settings export · spec: story.settings.export
- [ ] `wk-4` Crash when the report file is empty · type: defect

## In progress

- [ ] `wk-2` Bring the daemon target green · spec: story.adoption.target-add

## Blocked

## Done

- [x] `wk-1` Adopt SpecKit on the web target
```

Ids are `wk-<n>`, allocated as the next free number (`max(existing) + 1` —
never reused). Rendering is deterministic, so a `work` mutation produces a
minimal, reviewable diff.

The markdown provider deliberately has **no dependency graph** — typed
dependencies and a computed ready-queue are what the `beads` provider is for.

### `beads` — the `bd` dependency-aware tracker

```json
"work": { "provider": "beads" }
```

Shells out to the [Beads](https://github.com/steveyegge/beads) CLI, which must
be on `PATH` (a missing `bd` is a clear error naming the install). What Beads
adds over markdown: **typed dependencies** and Beads' own **ready predicate**
(`ready` = open ∧ unblocked, computed from the dependency graph), plus an
**atomic claim** (`bd update <id> --claim` — a compare-and-set, so two agents
can't take the same item).

State mapping: `ready` ↔ `open`, `in-progress` ↔ `in_progress`, `blocked` ↔
`blocked`, `done` ↔ `closed`. Other Beads statuses (e.g. `deferred`) pass
through as-is when reading; `move` and `list --state` accept only the canonical
four. Type mapping: `defect` ↔ Beads `bug`, `task` ↔ `task`. The `--spec`
pointer rides Beads' native spec-id field.

### `github-projects` — a Projects v2 board

```json
"work": { "provider": "github-projects", "project": 1, "owner": "acme" }
```

Work items are issues on a GitHub **Projects v2** board: `project` is the board
number, `owner` the board's owner (defaulting to the resolved repo's owner).
Auth is inherited from `gh` (`gh auth token`) — no token plumbing. This is
where defect intake on GitHub lives now: `specify work create "<title>"
--type defect`.

States map to the board's Status columns, overridable per invocation:

- `--status-field <name>` — the single-select field to drive (default `Status`).
- `--column <state>=<Column>` — repeatable, remaps one state (e.g.
  `--column ready=Todo`). Defaults: `ready`→`Ready`,
  `in-progress`→`In Progress`, `blocked`→`On Hold`, `done`→`Closed`.
- `--repo`, `--project`, `--owner` — override the resolved repo/board.

`claim` assigns you and moves the card in one step, and refuses an item already
assigned to someone else. Mutating verbs confirm before touching GitHub
(`--yes` skips the prompt) — the other providers are repo-local and never
prompt. The provider is deliberately flat: `ready` / `create` / `claim` /
`move` / `list`, nothing else.

### `none` — the surface disabled

```json
"work": { "provider": "none" }
```

Every `work` verb prints one line (`no work provider configured`) and exits 0 —
for projects that track work somewhere SpecKit shouldn't touch.
