# Commit Discipline

> Projected by `specify init` into the agent's rules dir and loaded every
> session.
>
> **Note for users who copied this template:** the default policy here is that
> the agent commits at natural commit points. If you'd rather hold commit
> boundaries yourself, override this with a feedback memory like `"Don't run git
> commits — I'll handle them"` and the agent will stop committing.

The agent commits often, atomically, and with messages a human would write.
Commits are how work becomes durable; treat them as a primary deliverable, not an
afterthought.

## When to commit

A commit lands at a **natural commit point** — a moment where the working tree
represents one coherent, internally consistent unit of work. Typical natural
commit points:

- A failing test was written, then the implementation made it pass, and the test
  now passes. (Two commits if the test-then-implementation distinction matters;
  one commit if they're tightly bound.)
- A self-contained refactor is complete and all tests still pass.
- A spec was edited and the affected tests/implementation re-checked.
- A configuration change was made and verified in the relevant tool.
- A feature folder was authored or extended and `specify scan` runs clean.

Do **not** commit mid-task work. If tests are red, if the file is half-edited, if
the change is incomplete — keep going (or stash and pivot). A WIP commit is the
wrong answer; finish the thought, then commit.

## What goes in one commit

**One logical change per commit.** If you can describe the commit as "X and also
Y", it's two commits.

Counter-examples that are _not_ one logical change:

- "Add the items list view model and also bump a dependency version"
- "Fix the duplicate-email bug and reformat the file"
- "Implement story.item.create and story.item.edit"

In each case, split. The dependency bump is its own commit. The reformat is either
its own commit or, ideally, dropped because it has nothing to do with the bug.

## What goes in a good message

A commit message has three parts: **subject**, optional **body**, optional
**trailer(s)**.

This repo uses **[Scoped Commits](https://scopedcommits.com/)**, not Conventional
Commits. The subject leads with the **scope** — the subsystem, area, or module
the commit touches — because in a spec-driven repo projected across many targets,
_where_ a change lands is the first thing a reader (or an incident responder)
needs to know.

### Subject

The shape is **`<scope>: <description>`**.

The **description**:

- Imperative, present tense: "add", "fix", "remove" — not "added" or "adds".
- The whole subject (scope included) stays under ~72 characters.
- No trailing period.
- Specific. "fix bug" is useless; "reject duplicate emails in item creation" is
  useful.

The **scope** names what the commit touches. Scoped Commits leaves the vocabulary
to the project; here a scope must be one of the **defined** scopes below — and
`specify gate scope` enforces it mechanically, rejecting a subject whose scope
isn't real. The set isn't a hand-maintained list: `specify` derives it from the
repo at commit time — every spec/feature `id:`, each target, each
`features/<NNNN>`, the harness areas — so adding a spec or a target makes that
scope usable automatically. Scopes that aren't derivable that way (a Go package
area, a custom workspace name) are declared, one per line, in
`.claude/commit-scopes`.

| Scope | Use for |
| --- | --- |
| a **spec / feature ID** — `story.item.create`, `domain.item` | A change scoped to one spec's behavior. The scope is a **reverse pointer to that `id:`** — same discipline as `// SPEC:`. |
| a **target** — any name defined under `targets` in `.speckit/specs.json` | Changes inside that target's tree (its configured `dir`). |
| `specs` | Cross-cutting spec files (`CONVENTIONS`, shared models). |
| `features/<slug>` | Authoring or extending a feature folder (slug must be a real `features/` directory). |
| a harness area — `hooks`, `skills`, `commands`, `agents`, `templates`, `rules`, `docs`, `readme` | Changes to the project's own machinery. |
| `treewide` | A genuinely repo-wide sweep with no single home. |

The IDs come straight from the `id:` frontmatter in `specs/` and `features/` —
list them with `grep -rhE '^id:' specs features`. When a change spans more than
one area, prefer the **broadest scope that still describes it**; only fall back to
a comma-separated list (`<target-a>, <target-b>: …`) when no single scope fits,
and to `treewide` for a true global sweep. A ticket number, when there is one,
goes in parentheses after the scope: `app (PROJ-12): …`.

Examples:

- `app: add items list view model with empty / loaded / error states`
- `story.item.create: reject item creation when email is already present`
- `specs: clarify duplicate-email handling in story.item.create`
- `templates: dispatch format-on-edit to the target's formatter`

Reverts, merges, and other mechanical commits don't have to follow this shape —
format them however is clearest.

### Body

Optional. Use it when the WHY isn't obvious from the subject. Wrap at ~72
characters. Explain the motivation, any non-obvious trade-offs, and anything a
future reader would want that the diff doesn't show. Skip it for trivial changes.

### Trailer(s)

Optional `Key: value` lines at the end of the message. Use for cross-references
(`Refs: story.item.create`), a ticket (`Ticket: PROJ-12`), breaking changes
(`BREAKING CHANGE: <description>`), or co-authorship.

## Staging

- **Never `git add .` or `git add -A`.** Both sweep up files you didn't intend —
  untracked artifacts, env files, build outputs. Stage by explicit path.
- **`git status` before staging.** Decide what belongs in this commit and stage
  exactly that.
- **Review the diff.** `git diff --staged` before committing. If something is in
  there you don't recognize or didn't intend, take it out.

## Pre-commit / gate checks

- The `specify gate` checks (`firewall`, `generated`, `scope`) keep each commit
  honest — wired as `pre-commit` / `commit-msg` hooks for fast local feedback.
- If a hook fails, **the commit didn't happen.** Fix the issue, re-stage, create a
  new commit. Do **not** `--amend` after a failed hook — there's nothing to amend.
- Never bypass with `--no-verify` unless the user explicitly asks. Hook failures
  are usually telling you something real.

## When to amend vs. new commit

- **Default: new commit.** Easier to reason about, revert, and review.
- **Amend only when:** the previous commit is local (not pushed), the new change
  is genuinely part of the same logical unit, and amending makes the history
  clearer (not just smaller).

## What never to commit

- Secrets (`.env`, credential files, API keys). If you see these in `git status`,
  stop and warn the user.
- Build outputs (`dist/`, `.output/`, `build/`). The `.gitignore` should exclude
  these — if it doesn't, fix the gitignore in its own commit.
- Personal IDE config (`.vscode/`, `.idea/`) unless the user explicitly asks.

## Frequency

Prefer **many small commits** over a few large ones. Five focused commits with
clear messages beat one giant "implement the items feature" commit every time.

## Push policy

Committing is one thing; pushing is another. **Do not push unless the user
asks** — even if commits are clean. The user controls when work goes upstream.
