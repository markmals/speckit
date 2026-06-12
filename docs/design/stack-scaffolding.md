# Design — stack scaffolding

**Status:** proposal, for review. No code yet.

## Motivation

Starting a new target means reassembling the same things every time: the right
packages at compatible versions, the build/test config, the project layout, and
— the SpecKit-specific part — the **scenario-binding test harness** that makes
`specify verify` work. We want `specify` to lay that down from a curated,
version-pinned template so the agent extends a working target instead of
bootstrapping one.

A **pack** (already shipped) is the agent's *guidance* for a stack. A
**scaffold** (this design) is the runnable *starter* for it. They're paired:
`specify target add` lays the starter and projects the pack.

This is **not** a generic `create-*`. The value is curating SpecKit's
*recommended* stack and baking in conventions a generic scaffolder can't —
chiefly the binding harness. (`~/Developer/Libraries/create-sprinkles` is the
prior art; we keep its template-scaffolder shape but use Go `text/template`, not
handlebars, and target SpecKit's stacks.)

## Command

```
specify target add <name> --stack <stack> [--dir <path>] [--product <p>] [--with <feature>...]
```

- `<name>` — the target's key in `.speckit/specs.jsonc` (e.g. `web`, `consumer-web`).
- `--stack` — which scaffold: `web` | `apple` | `android` | `windows` | `linux` | `go-cli` | `node-cli` | `rust-cli` | `website`.
- `--dir` — where to scaffold (default `apps/<name>`).
- `--product` — optional product label written onto the target.
- `--with` — optional add-ons the scaffold declares (e.g. `--with convex`).

**Flag-driven and non-interactive by default** — the primary caller is an agent,
which passes args. (Prompts for a human TTY can come later; not v1.) Lives under
a `target` command group so `target list` / `target remove` can follow.

What it does, in order:

1. Resolve `--stack` → load the scaffold manifest.
2. Collect template variables from flags + defaults; error on anything required and unset.
3. Render the template tree into `--dir`.
4. Compute the target's `specs.jsonc` entry from the manifest and add it (see *specs.jsonc merge* below).
5. Project the stack's pack (`specify packs`).
6. Print next steps (the install command, and `specify verify <name>`).

## Template layout (embedded in coreassets)

```
templates/scaffolds/<stack>/
├── scaffold.json          # manifest
└── files/                 # the template tree
    ├── package.json.tmpl  # *.tmpl → rendered through text/template, suffix stripped
    ├── vite.config.ts.tmpl
    ├── src/…              # non-.tmpl files copied verbatim
    └── tests/…            # the binding harness + one example spec test (green on arrival)
```

### Manifest (`scaffold.json`)

```jsonc
{
  "stack": "web",
  "delims": ["<<", ">>"],          // optional custom text/template delimiters (see below)
  "target": {                       // becomes the specs.jsonc entry ({{.Dir}}/{{.Name}} substituted)
    "command": "pnpm -C {{.Dir}} test --run",
    "format":  "junit",
    "report":  "{{.Dir}}/report.junit.xml",
    "source":  "{{.Dir}}/src"
  },
  "variables": [                    // beyond the always-present Name/Dir/Product
    { "name": "Module", "default": "{{.Name}}", "prompt": "package name" },
    { "name": "GithubOwner", "from": "git" }   // resolved from the git remote when omitted
  ],
  "features": {                     // optional --with add-ons
    "convex": { "files": "features/convex", "vars": { "Backend": "convex" } }
  }
}
```

## Rendering with `text/template`

- Files ending `.tmpl` are rendered through **`text/template`** (not `html/template`
  — these are code/config, not HTML) with the data context; the `.tmpl` suffix is
  stripped on output. Everything else is copied byte-for-byte.
- **Data context** (one struct, the same shape every scaffold sees):
  ```go
  type scaffoldData struct {
      Name, Dir, Product, Module, GithubOwner string
      Features map[string]bool   // {"convex": true}
      Vars     map[string]string // manifest-declared extras
  }
  ```
- **Custom delimiters.** Go's default `{{ }}` collides with what some target
  ecosystems already use in their own files (GitHub Actions `${{ }}`, some JS/Vue
  templating). Each scaffold may set `"delims"` in its manifest (e.g. `<< >>`) so
  the scaffold's templating never fights the files it emits. Default stays `{{ }}`.
- A small `FuncMap` for the obvious casings (`lower`, `title`, `kebab`, `pascal`)
  so one `Name` yields the package name, the Swift type name, etc.

## The binding-harness contract (the load-bearing part)

Every scaffold must leave the target **green on `specify verify` immediately** —
one example spec, one bound test, one `// SPEC:` pointer — so the agent extends a
working loop rather than wiring one. Concretely, per stack:

| stack | test framework → report `format` | binding affordance pre-wired | harness artifact |
| --- | --- | --- | --- |
| web / node-cli | Vitest → `junit` | `// [scenario.<id>]` above `it(…)` | vitest config emitting junit at `report` |
| apple | Swift Testing → `swift` (event-stream NDJSON) | `@Suite(.spec)` / `@Test(.scenario)` traits | `SpecTraits.swift` + the event-stream output flag |
| android | kotlin.test/JUnit → `junit` | `@Tag("spec:…")` / `@Tag("scenario:…")` | Gradle test task emitting junit |
| windows | MSTest → `junit` | `[TestProperty("scenario", …)]` | `.runsettings` emitting junit |
| linux / rust-cli | cargo-nextest / `go test` → `junit` | `// [scenario.<id>]` comment | nextest/gotestsum junit output |

Each ships an example `story.*` spec + the matching bound test, so `specify scan`
and `specify verify <name>` both pass on the freshly-scaffolded target.

## specs.jsonc merge (a real sub-decision)

Writing the new target into an existing `.speckit/specs.jsonc` while preserving
the user's comments and formatting is the awkward part — JSONC doesn't round-trip
through `encoding/json`. Options, simplest first:

1. **Absent → create; present → print the block to add.** `target add` writes the
   file if there's none, otherwise prints the ready-to-paste `"<name>": { … }`
   block and tells the user where to put it. Zero risk, a little manual.
2. **Append before the closing brace** of `targets` with a text edit (keeps
   comments, but brittle if the file is unusual).
3. **A comment-preserving JSONC editor** (round-trip via a CST). Correct, most work.

Recommend **(1) for the first slice**, with (2)/(3) as a later `--write` upgrade.

## Other open decisions

- **Dependency install.** v1 writes files and *prints* the install command
  (`pnpm install`, `swift build`) rather than running it — installers are slow,
  network- and environment-dependent, and better left to the user/agent.
- **Where targets live.** Default `apps/<name>`; `--dir` overrides. Worth a
  convention note in `docs/config.md` once it lands.
- **Interactivity.** Flags only for v1; TTY prompts later.

## First slice (when we build)

1. The `target add` command + the manifest/loader + the `text/template` renderer
   (custom delims, FuncMap, `.tmpl` handling) + specs.jsonc strategy (1).
2. The **web** scaffold end-to-end: a target that's green on `specify verify`
   the moment it's created.
3. Then **apple** — it exercises the other report format (`swift`) and the most
   distinct harness (`SpecTraits.swift`), proving the contract generalizes.

The remaining seven stacks are then mechanical (each is a manifest + a template
tree + the harness), parallelizable the way the packs were.
