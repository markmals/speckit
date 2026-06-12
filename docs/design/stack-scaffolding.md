# Design — stack scaffolding

**Status:** proposal. The three sub-decisions — plain `specs.json` (no JSONC),
keep `{{ }}` with escaping, and run the install — are resolved into this doc.
Awaiting final go-ahead to build.

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

- `<name>` — the target's key in `.speckit/specs.json` (e.g. `web`, `consumer-web`).
- `--stack` — which scaffold. **App stacks:** `web` · `website` · `apple` · `android` · `windows` · `linux` · `go-cli` · `node-cli` · `rust-cli`. **Library/extension stacks:** `swift-package` · `swift-cli` · `ts-lib` · `vscode-extension` · `browser-extension` — these set the product `kind: library` ([library-products.md](library-products.md)).
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
4. Compute the target's `specs.json` entry from the manifest and add it (load → add → write; see *specs.json merge* below).
5. Project the stack's pack (`specify packs`).
6. Run the scaffold's install (unless `--no-install`).
7. Print next steps (`specify verify <name>`).

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

`scaffold.json` is itself plain JSON. Its `target`/`variables` string values are
run through `text/template` (so `{{.Dir}}` etc. resolve):

```json
{
  "stack": "web",
  "install": "pnpm install",
  "target": {
    "command": "pnpm -C {{.Dir}} test --run",
    "format": "junit",
    "report": "{{.Dir}}/report.junit.xml",
    "source": "{{.Dir}}/src"
  },
  "variables": [
    { "name": "Module", "default": "{{.Name}}", "from": "flag" },
    { "name": "GithubOwner", "from": "git" }
  ],
  "features": {
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
- **Delimiters stay `{{ }}`; escape literals.** Where a template file contains a
  literal `{{` that isn't ours — chiefly a scaffolded GitHub Actions `${{ … }}` —
  escape it as `{{"{{"}}` (text/template's literal-string trick). Most files have
  no `{{` at all, so this is rare.
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
| swift-package / swift-cli | Swift Testing → `swift` | `@Suite(.spec)` / `@Test(.scenario)` traits | `SpecTraits.swift`; SwiftPM `swift test` (no simulator) |
| ts-lib | Vitest → `junit` | `// [scenario.<id>]` | tsup/vite-lib build; Vitest config (no dev server) |
| vscode-extension | `@vscode/test-cli` (Mocha) → `junit` | `// [scenario.<id>]` | the extension-host test runner emitting junit |
| browser-extension | Vitest / Playwright → `junit` | `// [scenario.<id>]` | web-ext test config emitting junit |

Each ships an example spec + the matching bound test, so `specify scan` and
`specify verify <name>` both pass on the freshly-scaffolded target. The
library/extension stacks set the product `kind: library` and lay a library layout
(no app shell) — see [library-products.md](library-products.md).

## specs.json merge (now trivial)

Because `specs.json` is plain JSON, it round-trips through `encoding/json`
cleanly — there are no comments to preserve. `target add` loads the config
(creating it if absent), adds the new target to `targets`, and writes it back.
No print-to-paste, no CST round-trip. (Key order isn't preserved across a
rewrite — fine for a generated config.)

## Dependency install

`target add` **runs** the scaffold's install (declared in the manifest —
`pnpm install`, `swift build`, …) so the target is ready to `specify verify`
immediately. It's slow and network-dependent, so a `--no-install` flag skips it
for offline / CI use.

## Other open decisions

- **Where targets live.** Default `apps/<name>`; `--dir` overrides. Worth a
  convention note in `docs/config.md` once it lands.
- **Interactivity.** Flags only for v1; TTY prompts later.

## First slice (when we build)

1. The `target add` command + the manifest/loader + the `text/template` renderer
   (FuncMap, `.tmpl` handling, escape literal `{{`) + the specs.json load→add→write + run-install.
2. The **web** scaffold end-to-end: a target that's green on `specify verify`
   the moment it's created.
3. Then **apple** — it exercises the other report format (`swift`) and the most
   distinct harness (`SpecTraits.swift`), proving the contract generalizes.

The remaining seven stacks are then mechanical (each is a manifest + a template
tree + the harness), parallelizable the way the packs were.
