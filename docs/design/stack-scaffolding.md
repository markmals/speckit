# Design — stack scaffolding

**Status:** built. Several stacks are green end-to-end — **web** (TanStack Start;
`--data`/`--runtime` variants + `--with` feature add-ons; UI from the racket-ui
shadcn registry), **go-service** (a Go HTTP daemon; members compose into one
repo-root `go.mod` via `sharedModule`; `--with openapi`/`sqlite`/`client`), **apple**
(a headless SwiftPM Core + a Tuist AppKit app + `--with swiftdata`/`openapi` + the
AppKit pack), and the two Swift **library** stacks **swift-package** (a flat reusable
library) and **swift-cli** (a library Core + a thin swift-argument-parser executable).
The library stacks reuse apple's headless harness with zero engine changes; the formal
product `kind: library` taxonomy ([library-products.md](library-products.md)) is still
pending (today their example specs use `kind: story`, which verifies fine). Beyond
the original decisions (plain `specs.json`, keep `{{ }}` with escaping, phased
post-render scripts that resolve versions by *running* `pnpm add`/`go get`), the
implementation added: **variants** (`--data`/`--runtime`, rendered over the base),
**features** (`--with`, rendered last), **`sharedModule`** (a repo-root module
shared by members), `Data.Module` (so a member imports its own generated/internal
packages by full path), and **`specify target register`** (onboard an existing
member — see below). The mechanics live in `internal/scaffold`; this doc is the
rationale.

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
specify target add <name> --stack <stack> [--dir <path>] [--product <p>] [--data <k>] [--runtime <k>] [--with <feature>...] [--no-install]
```

- `<name>` — the target's key in `.speckit/specs.json` (e.g. `web`, `consumer-web`).
- `--stack` — which scaffold. The roster is **evidence-based** — only stacks Mark has actually worked on (per `~/Developer` + the `markmals` / `markmals-archive` GitHub accounts). **App stacks:** `web` · `website` · `apple` · `android` · `go-cli` · `go-service` · `node-cli`. **Library stacks:** `swift-package` · `swift-cli` · `ts-lib` · `vscode-extension` — these will set the product `kind: library` once that taxonomy lands ([library-products.md](library-products.md)); the shipped swift stacks' example specs use `kind: story` in the meantime.
- `--dir` — where to scaffold (default `<memberDir>/<name>`, where `memberDir` is the manifest's placement — `apps` for web, `cmd` for go-service, `packages` for a library).
- `--product` — optional product label written onto the target.
- `--data` / `--runtime` — **variants**: selectable axes the manifest declares (e.g. web's `--data convex|drizzle|none` × `--runtime cloudflare|node`), each rendered *over* the base.
- `--with` — optional **features** (add-ons), rendered *last* (e.g. `--with openapi`, `--with clerk`).
- `--no-install` — skip the post-render scripts (offline / CI).

## Composition model (variants, features, shared modules)

Three layers render in order — **base → variants (`--data`/`--runtime`) → features
(`--with`)** — so later layers overwrite shared files (a data variant's `router.tsx`,
a feature's `main.go`) deterministically. Features are a Go map (non-deterministic
iteration), so two features must never write the same file; cross-cutting composition
goes through a **seam** instead (e.g. the web provider seam `app/providers.tsx`, where
each provider feature contributes a `{{if .Features.<name>}}` block). Each variant/
feature contributes its own deps (`pnpm add`/`go get`) and phased scripts.

**`sharedModule`** (manifest flag) makes a stack's members compose into ONE repo-root
module rather than a self-contained one each — go-service: `target add` writes a root
`go.mod` if the repo isn't a module yet (module path from the git remote, else the dir
name) and a second member just joins it. `Data.Module` carries that path into templates
so a member imports its own generated/internal packages by full path.

## Registering an existing member

```
specify target register <name> --stack <stack> [--dir <path>] [--format/--command/--report/--source/--bindings ...]
```

`target add` is for *new* members (it renders files + installs). To adopt SpecKit in a
repo whose code already exists, `target register` records the member as a target in
`.speckit/specs.json` **without** rendering or installing anything. It seeds the verify
wiring from the stack's scaffold manifest when one exists (web, go-service) via the same
`RenderTarget`; for stacks without a scaffold — or a member wired differently — the
fields come from flags (flags override the manifest defaults). This is the onboarding
path for converting a Workbench-shaped repo like trove.

**Flag-driven and non-interactive by default** — the primary caller is an agent,
which passes args. (Prompts for a human TTY can come later; not v1.) Lives under
a `target` command group so `target list` / `target remove` can follow.

What it does, in order:

1. Resolve `--stack` → load the scaffold manifest.
2. Collect template variables from flags + defaults; error on anything required and unset.
3. Render the template tree into `--dir`.
4. Compute the target's `specs.json` entry from the manifest and add it (load → add → write; see *specs.json merge* below).
5. Project the stack's pack (`specify packs`).
6. Run the scaffold's phased setup scripts (unless `--no-install`) — see *Post-render scripts* below.
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
  "scripts": [
    {
      "phase": 0,
      "commands": [
        "pnpm add react react-dom @tanstack/react-router @tanstack/react-start",
        "pnpm add -D vite vitest @typescript/native-preview @vitejs/plugin-react @tailwindcss/vite tailwindcss oxlint oxfmt oxlint-tsgolint eslint-plugin-perfectionist eslint-plugin-prefer-let",
        "pnpm install"
      ]
    }
  ],
  "target": {
    "command": "cd {{.Dir}} && mise run test",
    "format": "junit",
    "report": "{{.Dir}}/junit.xml",
    "source": "{{.Dir}}/app"
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
| web / website / node-cli / ts-lib | Vitest → `junit` | `// [scenario.<id>]` above `it(…)` | Vitest config emitting junit at `report` |
| apple / swift-package / swift-cli | Swift Testing → `swift` (event-stream NDJSON) | `@Suite(.spec)` / `@Test(.scenario)` traits | `SpecTraits.swift` (no simulator for package/cli) |
| android | kotlin.test/JUnit → `junit` | `@Tag("spec:…")` / `@Tag("scenario:…")` | Gradle test task emitting junit |
| go-cli | `go test` → `junit` | `// [scenario.<id>]` comment | gotestsum junit output |
| vscode-extension | `@vscode/test-cli` (Mocha) → `junit` | `// [scenario.<id>]` | the extension-host test runner emitting junit |

Each ships an example spec + the matching bound test, so `specify scan` and
`specify verify <name>` both pass on the freshly-scaffolded target. The
library/extension stacks lay a library layout (no app shell) and will set the
product `kind: library` once that taxonomy lands — see
[library-products.md](library-products.md). The shipped swift stacks already use
the library layout; their example specs use `kind: story` until then.

## Per-stack tooling previews (Mark signs off before each build)

The roster is evidence-based, but the **exact framework + tooling** for each
stack is *not* pre-decided here. The harness table above fixes only the contract
(report `format` + the binding affordance — SpecKit's convention). The actual
stack — web framework (TanStack Start vs Solid Start vs …), the `ts-lib` bundler,
the `apple` minimum-OS / Swift version, the Vitest/test-runner versions — is
chosen at build time, targeting the **latest / bleeding-edge** of each ecosystem,
and **previewed for Mark's sign-off before that stack's scaffold is built**. Each
preview is a short list of the pinned deps + the layout; approved choices are
recorded per stack as we go. No scaffold is built on a stack until its tooling is
approved.

## specs.json merge (now trivial)

Because `specs.json` is plain JSON, it round-trips through `encoding/json`
cleanly — there are no comments to preserve. `target add` loads the config
(creating it if absent), adds the new target to `targets`, and writes it back.
No print-to-paste, no CST round-trip. (Key order isn't preserved across a
rewrite — fine for a generated config.)

## Post-render scripts (resolve by running — don't freeze versions)

After the file tree is written, `target add` runs the manifest's **`scripts`** in
the freshly-rendered target dir, so the target is ready to `specify verify`
immediately. Each script is `{ "commands": [...], "phase": N, "silent": bool }`;
phases run in ascending order, commands within a script in sequence, and a
`silent` step's failure (and output) is swallowed — for best-effort steps like a
codegen that needs a prior login. A `--no-install` flag skips the whole run for
offline / CI use.

The load-bearing principle (borrowed from create-sprinkles): **resolve values by
running the tool, never hardcode them into a template.** The canonical case is
dependency versions. The template's `package.json` carries *no* dependency
block; a phase-0 `pnpm add …` makes the package manager resolve each dependency
to its current latest and pin it (`"vite": "^8.0.16"`) into `package.json` at
scaffold time. A template that hardcoded `"latest"` would be worse than useless:
`pnpm install` does **not** rewrite `"latest"`, so the project would float on
every future install. The same mechanism carries framework codegen
(`react-router typegen`, `wrangler types`, `convex dev --once`) and formatting —
anything whose correct value depends on the installed toolchain or the registry,
not on the template author.

Phases, by convention (mirrors create-sprinkles):

| phase | purpose | example |
| --- | --- | --- |
| 0 | resolve + install dependencies | `pnpm add …`, then `pnpm install` |
| 1 | codegen that needs the deps | `wrangler types`, route-tree typegen |
| 2 | format the generated code | the stack's formatter |
| 3 | feature-specific setup (often `silent`) | `convex dev --once` |

Non-JS stacks use the same shape with their own tools (`swift build`,
`./gradlew build`, …).

## Other open decisions

- **Where targets live.** Default `apps/<name>`; `--dir` overrides. Worth a
  convention note in `docs/config.md` once it lands.
- **Interactivity.** Flags only for v1; TTY prompts later.

## Build status & next slices

1. ✅ The `target add` command + the manifest/loader + the `text/template`
   renderer (FuncMap, `.tmpl` handling, escape literal `{{`) + the specs.json
   load→add→write + the phased post-render script runner; plus **variants**
   (`--data`/`--runtime`), **features** (`--with`), **`sharedModule`**, and
   **`target register`**.
2. ✅ The **web** scaffold end-to-end — `{cloudflare,node} × {convex,drizzle,none}`
   + `--with` clerk·tiptap·posthog·email·stripe, UI from the racket-ui shadcn
   registry — green + locked the moment it's created.
3. ✅ The **go-service** scaffold — a Go HTTP daemon in `cmd/<name>`, members
   sharing one repo-root `go.mod`, `--with openapi`/`sqlite`/`client` (the
   trove-shaped contract-first + persistent + external-client service).
4. 🟡 The **apple** scaffold — exercises the other report format (`swift`,
   event-stream NDJSON) and the most distinct harness (`SpecTraits.swift`'s
   `.scenario()` trait), proving the contract generalizes. **Slice 1 shipped**:
   the headless SwiftPM `Core` is the verify target (no Tuist/Xcode/simulator/
   signing needed), green-on-arrival on the Xcode toolchain alone. Remaining
   slices (Tuist app surface, `--with` features, the AppKit pack) in
   [scaffolds/apple.md](scaffolds/apple.md). Then **ts-lib** + **go-cli** (the
   remaining trove member shapes).

The remaining stacks follow one at a time — each gated on its tooling preview
(above), then a manifest + template tree + harness, parallelizable the way the
packs were.
