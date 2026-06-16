# Mise monorepo adoption

How a SpecKit repo becomes a real [mise monorepo](https://mise.jdx.dev/tasks/monorepo.html):
a generated root `mise.toml` that unifies the task namespace across every
`specify target add`ed member, hoists shared toolchains, and DRYs member tasks
behind per-family `task_templates`.

Status: **design** (not yet built). Owner: Mark. Date: 2026-06-16.

## Problem

SpecKit already composes multi-target repos: `specify target add <name> --stack
<s>` renders a member into `apps/<name>` (or `cmd/<name>`, `packages/<name>`),
registers it in `.speckit/specs.json`, and the engine verifies it. But it does
**not** use mise's monorepo-tasks feature:

- Each member ships its own self-contained `mise.toml` — re-pinning the same
  toolchain (`node`, `pnpm`, `gh`, `1password`) and repeating the same
  `test`/`fmt`/`lint` task bodies across members of the same family.
- There is no root config, so there is no unified namespace: you cannot
  `mise //apps/web:test` or `mise //...:test` or `mise tasks --all` from the
  repo root.
- The engine verifies a member with `cd {{.Dir}} && mise run test` — a subshell
  `cd`, not the native monorepo invocation.
- Trust is per-member.

mise (verified on 2026.6.6) supports all the pieces we want: `monorepo_root =
true` + a **required** `[monorepo].config_roots`, target-path syntax
(`//dir:task`, `//...:task`), `[task_templates]` with `extends`, tool/env/`vars`
layering, and **root-trust propagation**. (`config_roots` is now required —
filesystem auto-discovery is deprecated and warns.)

**Goal:** from the first `target add`, a SpecKit repo is a real mise monorepo.

## Decisions (from the brainstorm)

1. **Root config from the first target.** Every member lives in a subdir, so a
   root config is useful immediately; the invariant "a SpecKit repo is a mise
   monorepo" is simpler than "becomes one at member #2."
2. **`config_roots` as memberDir globs** — `apps/*`, `cmd/*`, `packages/*` — not
   an explicit per-target list.
3. **Hoist tools to root always; hoist task *bodies* lazily — at member #2.**
   `monorepo_root`, `config_roots`, and the toolchain `[tools]` land in the root
   config from the first member (a single root `[tools]` has no indirection
   cost). A family's `[task_templates]` are written only once the family has a
   **second** member; until then the lone member keeps its task bodies **inline**
   and self-contained, so a single-member repo reads cleanly. Families (not
   individual stacks) share templates: a website and a web app; an iOS app and a
   Mac app; a Go daemon and a Go TUI.
4. **Engine verifies via `mise //<dir>:<task>`** — the native monorepo form.
5. **The root config is edited with a real TOML library** (`go-toml/v2`),
   doing a **comment-preserving surgical merge** — no `# >>> speckit … # <<<`
   managed region, no clobbering user edits.

## Empirical validations

So the spec is not guesswork — each was reproduced against mise 2026.6.6 /
go-toml/v2 v2.3.1 before writing this:

| Claim | Result |
| --- | --- |
| `mise //apps/web:test` from repo root runs with **cwd = the member dir**; a relatively-written `junit.xml` lands in `apps/web/`. | ✅ — so the engine's `report: {{.Dir}}/junit.xml` (relative to root) still resolves. |
| A root `task_template` + member `extends` + member `[vars]` interpolates via mise's Tera `{{ vars.x }}`. | ✅ — `apple:build` with `run = "… {{ vars.scheme }}"` ran `scheme=FooApp` from the member's `[vars]`. |
| One `swift:fmt` walking the **dir superset** (`Core macOS iOS Sources Tests`) with `[ -d ]` guards works across both the apple and swift-package layouts. | ✅ — printed only the dirs that exist in each member. |
| `go-toml/v2`'s `unstable` parser exposes `Node.Raw` byte ranges, enabling a splice that adds a `config_roots` glob + a `[tools]` key + a new table while **preserving every comment**; output re-parses. | ✅ — see [the merge engine](#the-merge-engine-comment-preserving). |
| `1password` is the canonical mise registry name. | ✅ — `1password`, `1password-cli`, and `op` all alias `aqua:1password/cli`; the shipped `op = "2"` works today, but `1password` is canonical and drift-proof. The **binary** stays `op`. |

## Architecture

### Families

Each stack maps to a **family** (a new `family` field in `scaffold.json`). A
family owns one contribution — its toolchain pins and its `[task_templates]` —
and every stack in the family reuses it.

| Family | Stacks | Members that share templates |
| --- | --- | --- |
| `node` | `web`, `node-cli` | website, web app, CLI |
| `swift` | `apple`, `swift-package`, `swift-cli` | iOS/Mac app, library, CLI |
| `go` | `go-service` (+ future `go-tui`) | daemon, TUI |

### Family contribution files

`internal/coreassets/templates/monorepo/<family>.toml` — a parseable TOML
fragment authored once per family, holding:

```toml
# node.toml
[tools]
node = "24"
pnpm = "11"
gh = "2.94"
1password = "2"

[task_templates."node:test"]
description = "run Vitest with the junit reporter the engine joins"
run = "vitest run --reporter=junit --outputFile=junit.xml"

[task_templates."node:fmt"]
run = "oxfmt app"
# … routes, dev, typecheck, fmt:check, lint, build
```

For name-parameterized tasks the body uses mise's own Tera vars (the fragment is
**raw** — never run through Go's `text/template`, so `{{ vars.x }}` survives):

```toml
# swift.toml
[task_templates."swift:build-app"]
run = "tuist generate --no-open && tuist xcodebuild build -scheme {{ vars.scheme }}"

[task_templates."swift:fmt"]
run = '''
for d in Core macOS iOS Sources Tests; do
  [ -d "$d" ] && swift format --in-place --recursive "$d"
done
'''
```

### The root config

`mise.toml` at the repo root, created on the first `target add` and merged on
each subsequent one:

```toml
# Managed by `specify target` — your edits and comments are preserved.
# Task bodies live in the [task_templates] below; per-member config lives in each member's mise.toml.
monorepo_root = true

[monorepo]
config_roots = ["apps/*", "cmd/*"]   # one <parent>/* glob per target dir, deduped + sorted

[tools]                              # union of the present families' toolchains
node = "24"
pnpm = "11"
gh = "2.94"
1password = "2"
go = "1.26"

# ---- node family (written only once the node family has ≥2 members) ----
[task_templates."node:test"] …
# ---- go family ----
[task_templates."go:test"] …
```

`monorepo_root`, `[monorepo]`, and `[tools]` are present from the first member;
a family's `[task_templates]` block appears only when that family gains its
second member (see [promotion](#promotion-the-member-1-conversion)).

It is **partly managed, partly the user's**: SpecKit only ever *adds* its
managed entries (idempotent add-if-absent); the user may add their own
`[tasks.*]`, `[env]`, extra tools, and comments anywhere, and they survive every
future `target add`.

### Member configs: inline first, hoisted at member #2

A member's `mise.toml(.tmpl)` always drops `[tools]` (inherited from root). What
happens to its **task bodies** depends on how many members its family has:

**One member (inline, self-contained).** The lone member keeps full task bodies;
no family `[task_templates]` exist yet. This is what a single-member repo — the
common case — reads as:

```toml
# apps/<name>/mise.toml  (node, family has one member)
[env]
_.path = ['{{config_root}}/node_modules/.bin']

[tasks.test]
description = "run Vitest with the junit reporter the engine joins"
run = "vitest run --reporter=junit --outputFile=junit.xml"

[tasks.typecheck]
depends = ["routes"]
run = "tsgo --noEmit"
# … routes, dev, fmt, fmt:check, lint, build — all inline
```

**Two+ members (hoisted).** Adding the family's second member triggers
**promotion**: the family's `[task_templates]` are written to the root config,
the new member is rendered with `extends`, and the existing member's tasks are
converted in place. Members then read:

```toml
# apps/<name>/mise.toml  (node, family has ≥2 members)
[env]
_.path = ['{{config_root}}/node_modules/.bin']

[tasks.test]
extends = "node:test"

[tasks.typecheck]
extends = "node:typecheck"
depends = ["routes"]          # member-specific keys survive the conversion
# … routes, dev, fmt, fmt:check, lint, build
```

```toml
# <member>/mise.toml  (swift/apple, ≥2 members)  — vars feed the shared templates
[vars]
scheme = "{{pascal .Name}}"

[tasks.build]
extends = "swift:build-app"
```

So the DRY payoff arrives exactly when there is something to share (web app +
website, iOS + Mac), and never imposes `extends` indirection on a family of one.

### Promotion (the member-#1 conversion)

When `target add` brings a family from one member to two, it must convert the
**first** member's inline tasks to `extends`. The rule keeps it safe:

- Convert a `[tasks.X]` to `extends = "family:X"` **only if its body still equals
  the canonical scaffolded body** (the family template's `run`, after the
  member's `vars` substitution). A task the user has edited is **left inline** —
  never silently replaced by the shared template.
- Member-specific keys (`depends`, extra `env`) are preserved; only `run`/
  `description` collapse into the `extends`.
- The rewrite uses the same comment-preserving surgical merge as the root config,
  so the member's comments and any custom tasks survive.

Promotion keys on **family** membership, not stack: `target add web` then
`target add … --stack node-cli` is two members of the `node` family and triggers
it, as does a second `web`. A family's templates cover the tasks its stacks share
(identical, or unifiable via `vars`/superset-walk); tasks that genuinely differ
per stack within a family stay inline or get stack-scoped template names.

### Engine verification

Each `scaffold.json` `target.command` changes from `cd {{.Dir}} && mise run
test` to `mise //{{.Dir}}:test`. `report` (`{{.Dir}}/junit.xml`) and `source`
(`{{.Dir}}/app`) are unchanged — validated above, the task runs with cwd = the
member dir so the relative report lands where the engine reads it. No engine Go
change is required (the command is data in `specs.json`); only the comment in
`internal/engine/verify_run.go` is refreshed. **Existing** repos keep their
recorded `cd …` command (still valid mise); only newly-added targets get the
target-path form.

### Trust

The scaffold's phase-0 `mise trust` runs at the repo root once the root config
exists; root trust propagates to all descendant configs, so the per-member trust
step disappears.

## The merge engine (comment-preserving)

`go-toml/v2` (like every marshal-from-struct TOML library) drops comments on a
read-modify-write round-trip, and we will not use a `# >>> speckit` managed
region. Instead we edit **surgically** with the library's `unstable` parser,
which exposes each AST node's `Raw Range` (byte offset + length into the source).
We compute insertion points and splice into the *original bytes* — so every byte
we do not touch (all comments, all user content, all formatting) is preserved
verbatim.

`internal/scaffold/monorepo.go` (new):

```go
// EnsureRootMise creates or merges the repo-root mise.toml so it declares
// monorepo_root, the config_roots globs for every target dir, and the union of
// the present families' [tools]. A family's [task_templates] are written only
// once it has two members (promotion). Idempotent: re-running adds nothing.
// Preserves all existing comments and user content.
func EnsureRootMise(root string, families []Family, targetDirs []string) (changed bool, err error)
```

Merge operations, each via a node `Raw` byte-range splice:

- **Create** (no root `mise.toml`): render the managed skeleton from a
  `text/template` (with the header comments) + concatenate the present families'
  `[task_templates]` blocks verbatim.
- **`monorepo_root`**: splice `monorepo_root = true` if the key is absent.
- **`config_roots`**: for each target dir, ensure `path.Dir(dir)+"/*"` is an
  element of the array; splice `, "glob"` before the array's closing `]` if
  missing. (A dir at the repo root — parent `.` — falls back to an explicit
  entry; the default memberDirs prevent this in practice.)
- **`[tools]`**: splice each missing family tool key after the table's last
  key/value (never overwrites a user-pinned version).
- **`[task_templates."family:*"]`**: at a family's second member, append the
  family's blocks verbatim at EOF if not already present.

The same byte-range splice runs in the other direction on a **member** file
during [promotion](#promotion-the-member-1-conversion): replace a `[tasks.X]`
node's `run`/`description` with `extends = "family:X"` when the body is still
canonical, leaving the rest of the member's config (comments, `depends`, custom
tasks) untouched. So one engine covers both the root merge and the member
conversion.

This was prototyped end-to-end (add a glob, add a tool, append a table) against
a hand-commented document — all comments survived and the result re-parsed.

`go.mod` gains `github.com/pelletier/go-toml/v2` (the repo has no TOML dependency
today). The `unstable` package is, by name, unstable across go-toml versions —
so we pin the version and cover the merge with round-trip tests.

## Implementation plan (vertical slices by family)

Each slice is one family taken end-to-end — contribution file, inline member
templates + the promotion conversion, generator wiring, golden regen — so every
stage is independently testable and Stage 1 de-risks the whole mechanism.

- **Stage 1 — mechanism + `node` family.** Add `go-toml/v2`; build the surgical
  merge engine (root `EnsureRootMise` + member promotion) + the `Family` model +
  `family` in `scaffold.json`; author `monorepo/node.toml`; render `web` and
  `node-cli` members **inline** with `[tools]` hoisted to root; wire promotion
  into `target add` / `target register` (write `node:*` templates + convert the
  first member when the family reaches two); switch the node stacks'
  `target.command` to `mise //{{.Dir}}:test`; `mise trust` the root. This stage
  exercises both the one-member (inline) and two-member (hoisted) paths. Golden
  regen.
- **Stage 2 — `swift` family.** `monorepo/swift.toml` with `[vars]`-parameterized
  build/launch/run tasks and the superset-walk `swift:fmt`/`swift:lint`; inline
  members + promotion for `apple`, `swift-package`, `swift-cli`. Golden regen.
- **Stage 3 — `go` family.** `monorepo/go.toml`; inline + promotion for
  `go-service` (keep the `--with openapi` `generate` task; `SharedModule` go.mod
  handling is unchanged). Golden regen.
- **Stage 4 — docs + memory.** Update `docs/design/scaffolds/web.md` and
  `node-cli.md` to the current API (required `config_roots`, target-path
  invocation, deprecation of auto-discovery, `task_templates`); cross-link this
  doc; add a `.claude/memory/` topic for the generated-root-config invariant and
  the `unstable`-parser merge gotcha.

## Risks & caveats

- **Promotion mutates the first member's config.** Bringing a family to two
  members rewrites the first member's task table from inline bodies to `extends`.
  It only touches tasks whose body is still the canonical scaffolded one (a
  user-edited task is left inline, never silently overridden) and runs through
  the same comment-preserving surgical merge — but it is the main added
  complexity of the inline-first choice over always-hoist.
- **Generated-but-merged root config.** SpecKit only adds (never removes) its
  managed entries; a user who deletes one gets it re-added on the next `target
  add` (idempotent add-if-absent). User comments/tasks/tools are preserved by the
  surgical merge.
- **`go-toml/v2` `unstable` API** may shift between library versions — pinned +
  test-covered.
- **Golden churn** across every scaffold — expected; run the `mise run ci` gate +
  golden regen per the dev workflow.
- **Two templating layers** (`Go text/template` at scaffold time, mise Tera at
  run time) coexist because the family fragments are raw (never Go-rendered) and
  the member `.tmpl`s are — they never share a file.

## Testing

- **Unit (root merge):** create-from-empty; merge-into-existing (add glob / tool
  / family templates); **idempotency** (second run is a no-op); **comment
  preservation** (a fixture with user comments + a hand-added `[tasks.*]` round-
  trips untouched); multi-family composition; the `path.Dir(dir)+"/*"` glob
  derivation incl. a custom `--dir`.
- **Unit (promotion):** a one-member family stays inline; the second member
  writes `family:*` templates and converts the first member to `extends`; a
  member task the user **edited** is left inline (not overridden); `depends` and
  custom tasks/comments survive the conversion; promotion is idempotent.
- **Golden:** updated scaffold outputs for each family, in both the one-member
  (inline) and two-member (hoisted) shapes.
- **Integration/e2e:** scaffold a `web` member, run `mise //apps/web:test`,
  assert green + `apps/web/junit.xml` (inline path); add a second `node` member
  and assert the `node:*` templates resolve, the first member now `extends` them,
  both verify green, and `config_roots` still reads `apps/*` (the one glob covers
  both).
