# Apple scaffold — working knowledge

The `apple` stack scaffold (`internal/coreassets/templates/scaffolds/apple/`).
Design: [`docs/design/scaffolds/apple.md`](../../docs/design/scaffolds/apple.md).
Baseline + reference repos: gourmand (Tuist app, the convert target),
apple-platform-tools (raw SwiftPM, the binding reference), mac-dev-skills (the
AppKit agent pack). Built in slices; **Slices 1–2 (headless Core + Tuist app) + Slice 3
(`swiftdata`, `openapi`, `dist`→app-store-connect deploy kind) + Slice 4 (the AppKit pack)
shipped 2026-06-14**.

**Slice 4 — the pack** (`templates/packs/apple/`): 14 AppKit `SKILL.md` adapted from
mac-dev-skills (Mark's own, MIT) into SpecKit's concise pack style, beside the existing
`ios-development`/`ios-simulator-control` (16 total). **Pack projection is directory-driven**
(`internal/project/pack.go` scans `templates/packs/<stack>/*/SKILL.md`) — add a dir, zero Go
changes; pack skills aren't in `init`, so `TestInitGoldenTrees` never drifts (only
`templates/skills/` global skills do). Packs project **skills only** (no agents) — the source
`appkit-dev` agent's grounding mandate folded into `appkit-design`. **SpecKit deliberately does
NOT project MCP config** — per-machine tools (Xcode MCP) live in user `~/.claude`/`.mcp.local.json`,
documented in-skill. Adapt, don't vendor (point at first-party docs + sdk-api/sdk-search from
apple-platform-tools; never the 230 HIG files or tool binaries). **Pack-agent gating FIXED (#37):**
`specify init --integration <agent>` now seeds `agent` in `.speckit/specs.json` (`config.SetAgent`,
called from `project.Init`), so `target add`/`packs` auto-project; `AddTarget` preserves it.

## The things that cost real time

- **Verify target = the headless SwiftPM `Core`**, not the Tuist app. `swift test`
  on the pure package is headless/deterministic — no Tuist, Xcode project,
  simulator, or signing. That's what makes `apple` green-on-arrival in plain CI.
  The Tuist app surface (Slice 2) is the buildable product; its AppKit tests are a
  Mac-only secondary, never the gate.

- **The exact swift report command** (Swift 6.4): the event-stream flags are
  hidden from `--help`. Use:
  `swift test --package-path Core --event-stream-output-path test.swift-events.ndjson --event-stream-version 0`.
  `--event-stream-output-path` resolves **relative to `--package-path`**, so from
  the member root that path lands the report at `Core/test.swift-events.ndjson`.
  The engine joins it via `internal/reports/events.go:ParseSwiftEvents`.

- **Directory names can't be templated.** `scaffold.renderSubtree` substitutes file
  *contents* and strips `.tmpl`, but copies dir/file *names* verbatim. So you can't
  have a `Sources/{{.Name}}Core/` dir. Trick: keep the dir static (`Sources/Core`,
  `Tests/CoreTests`) and give the module a dynamic name via an explicit SwiftPM
  `path:` — `.target(name: "{{pascal .Name}}Core", path: "Sources/Core")`. Yields
  `GourmandCore`-style names with a fixed directory.

- **`.tmpl` is for substitution ONLY — the inverse of the go-service rule.**
  speckit's Go/Linux CI never compiles Swift (the go-service base `files/` IS
  compiled, which forces its `.tmpl`-for-nonstdlib discipline; that does NOT apply
  here). A Swift file is `.tmpl` iff it contains a template var: `Package.swift.tmpl`
  + `TodoTests.swift.tmpl` (import the module name); `SpecTraits.swift`, `Todo.swift`,
  `TodoList.swift` are plain `.swift`.

- **Binding is the `.scenario()` trait** (decided 2026-06-14, canonical), matching
  apple-platform-tools and the engine's `swiftBindRe`:
  `@Test(.scenario("scenario.x")) func \`readable name\`()` — NO display string; the
  raw-identifier function name IS the event-stream `displayName`, which is the join
  identity. gourmand's legacy `@Test("[scenario.x] …")` display-name form is NOT
  canonical; converting gourmand migrates its tests to the trait.

- **Scaffold scripts run with cwd already in the member dir** (`runIn(dir, …)` in
  `cmd/specify/main.go`), so a phase script is bare `mise trust`, NOT
  `cd {{.Dir}} && mise trust` (double-cd → fails). But the target `command` stored
  in specs.json IS run from the project root, so it keeps `cd {{.Dir}} && mise run test`.

- **swift-format**: ship gourmand's `.swift-format` (lineLength 100, 4-space,
  `lineBreakBeforeEachArgument: true`); without a config swift-format defaults to
  2-space and lint fails. To author lint-clean templates: write the file, run
  `swift format --in-place` against that config, then templatize the formatted output.

## Slice 2 (Tuist app surface) gotchas

- **Tuist anchors on the repo root** (closest `.git` or `Tuist/` dir) — `tuist
  generate` fails outside a git repo. Fine in practice (every SpecKit project is a
  git repo); the e2e must `git init` the throwaway project before building the app.
- **The Core package needs a `.library` product** for the Tuist app to consume
  (`.package(product: "<Name>Core")`). Slice 1 shipped none (swift test joins the
  test target to the source target directly); added to `Package.swift.tmpl` —
  forward-compatible, swift test ignores it.
- **Import order is name-dependent** → a **silent phase-1 `swift format --in-place`**
  pass normalizes it. `OrderedImports` sorts lexicographically, but `import AppKit`
  vs `import <Name>Core` flips with the name (`AppCore` < `AppKit` < `GourmandCore`),
  so no fixed template order is clean for all names. Run `swift format` DIRECTLY in
  the script (not `mise run fmt`) so it doesn't drag a Tuist install into `target add`;
  use the `if [ -d ]; then … fi` form so the loop exits 0 when `iOS/` is absent.
- **`@main` for AppKit** = `@main @MainActor final class AppDelegate: NSObject,
  NSApplicationDelegate` with an explicit `static func main()` (programmatic; the
  default delegate `main()` calls `NSApplicationMain`, which needs a storyboard).
- **`verify` does NOT pull Tuist** — `test` is `swift test` only; the `[tools]` tuist
  pin only fetches on the first app task (`generate`/`build`). Proven: verify ran in
  ~7s with no download.
- App-level tests (`macOS/Tests`, `tuist test`) are a **Mac-only secondary**, never
  the verify gate — verify always targets the headless Core.

## Slice 3 (`--with` features) gotchas — established by `swiftdata`

- **`Package.swift` is the composition seam.** Features render in nondeterministic
  map order and must never write the same shared file, but SPM deps/targets are
  declarative in the manifest. So the BASE `Package.swift.tmpl` carries each feature's
  target/product/deps behind `{{if .Features.<name>}}`; the feature ships ONLY additive
  source files (new dirs). This is the web `providers.tsx` seam, not the go-service
  `go get`-script approach (SPM has no `go get` equivalent). The example story
  (`root/.../todo.manage.md.tmpl`) is a seam too — features add scenarios via the same
  `{{if}}` gate. (Also: name the story file `.md.tmpl`, not `.md`, or `{{.Dir}}`/`{{if}}`
  render literally — a Slice-1 bug fixed here.)
- **ONE test target, always.** `swift test --event-stream-output-path` has each test
  TARGET's process truncate-write the same file → **multiple test targets clobber each
  other's events**, the engine joins only one, and the rest show `unjoinable`/`dangling`.
  Fix: a feature's tests live in the single `CoreTests` target (which gains a conditional
  dependency on the feature's source target via the seam), never a target of their own.
- **Shared `TestSupport` target.** The `.spec`/`.scenario` traits moved from inside
  `CoreTests` to a shared `Tests/Support` library target (`TestSupport`, public types),
  imported by every test target — apple-platform-tools' pattern. Required once a feature
  test needs the traits too.
- **SwiftData specifics**: a `<Name>Persistence` target with `.strictMemorySafety()`
  OFF (SwiftData macros are outside the proof); test headlessly with a temp-file
  `ModelConfiguration(url:)` reopen (real persistence proof, no app/simulator). The
  feature test uses plain `import <Name>Persistence` (public API), not `@testable`.
- **OpenAPI specifics** (`--with openapi`): the **Swift OpenAPI Generator** build-tool
  plugin generates `Client`/`Types` from `Sources/API/openapi.yaml` at build time.
  - **SwiftPM arg order**: package `dependencies:` must come **after `products:` and
    before `targets:`** — wrong order → "argument 'products' must precede 'dependencies'".
  - Generated code is `internal` (config `accessModifier: internal`); expose a public
    hand-written facade (`TodoAPIClient`) that maps the wire schema to the domain — the
    test only touches the facade + the public `ClientTransport`.
  - Test offline with a **fake `ClientTransport`** returning canned JSON (the
    httptest-fakeServer analog). Needs `import HTTPTypes` (transitive via OpenAPIRuntime).
  - First build resolves swift-openapi-* (network, ~35s cold) + runs codegen; cached
    after. Changing the contract may need a clean build to bust the codegen cache.
  - The example is reshaped around `Todo` (fetch `[Todo]` from `/todos`) so it stays in
    the existing story (`scenario.todo.manage.fetch`), like swiftdata's persist scenario.

## Distribution — a deploy kind, not a `--with` feature

`dist` ships as the **`app-store-connect`** deploy kind (`config.DeployKinds` +
`templates/deploy/app-store-connect/deploy.yml.tmpl`), NOT a scaffold feature —
scaffold features render only into the member dir, but release workflows belong at the
repo-root `.github/`, which the deploy subsystem (`specify deploy add`) owns. The
workflow archives with `xcodebuild` and uploads via the **`asc`** CLI
(github.com/rorkai/App-Store-Connect-CLI, `brew install asc`; `asc auth login
--bypass-keychain` + `asc builds upload --archive-path … --workspace … --scheme …`) on a
`v*` tag. Secrets are op:// refs (`ASC_KEY_ID`/`ASC_ISSUER_ID`/`ASC_API_KEY_BASE64` +
the cert/keychain trio); the app's ASC id is a non-secret **repo variable** `ASC_APP_ID`
(`${{ vars.ASC_APP_ID }}`), not a secret. Deploy templates use `[[ ]]` delims (so `${{ }}`
survives) + the `pascal`/`kebab` funcs; `[[pascal .Name]]` = the Xcode scheme. Validated
by `TestRenderDeployAllKinds` (auto-covers every kind) + actionlint. `push` (APNs) was
deferred (config-heavy, no offline test, gourmand has no server). Future kinds:
Developer-ID-notarize, Homebrew.

- **Structure-only Go test**: `internal/scaffold/apple_test.go` asserts the rendered
  tree + the dynamic-module/static-dir wiring + story↔`.scenario` id agreement. The
  Swift build/verify is proven by a Mac e2e (`target add` → `verify`), never by
  speckit's CI.

## Sibling library stacks — `swift-package` / `swift-cli` (shipped)

Two sibling library stacks that reuse the apple harness with **zero engine
changes** (`internal/coreassets/templates/scaffolds/swift-{package,cli}/`). Their
example specs use `kind: story` (the formal product `kind: library` taxonomy is still
pending — [[library-products]]), so nothing here sets `kind: library` yet. Both:
`memberDir: packages`, `format: swift`, `bindings: scoped`, the shared `TestSupport`
target with the `.spec`/`.scenario` traits, and the event-stream `mise run test` task.
Verify target = the package itself (`swift test`), green-on-arrival with only the
Swift toolchain (no Tuist/Xcode). Render-tested: `internal/scaffold/swift_{package,cli}_test.go`.

- **`swift-package`** — a FLAT library (no `Core/` subdir): module `{{pascal .Name}}`
  over static `Sources/Library` via `path:`; test target over `Tests/LibraryTests`.
  Example is a pure `SemanticVersion` bound to `story.version.compare` — chosen
  DISTINCT from apple's `todo` so two stacks added to one repo never collide on
  `features/` or duplicate a story id.
- **`swift-cli`** — library Core + thin executable shell. `{{pascal .Name}}Core`
  (static `Sources/Core`) holds the provable logic; a `{{kebab .Name}}`
  `executableTarget` (static `Sources/CLI`, `@main`/`ParsableCommand`) parses args and
  delegates. **swift-argument-parser** is an unconditional package dep — so its
  `dependencies:` block sits after `products:` (the SPM arg-order rule). The behaviour
  is proven through the Core lib, so `swift test` verifies headlessly; the binary is
  never run during verify (but `mise run run` exists; e2e: `greet-tool Ada --shout` →
  `HELLO, ADA!`). Example bound to `story.greet.run`.

**Packless stacks are first-class** (`internal/project/pack.go`): neither ships a pack
(library stacks have no platform skill suite). `loadPack` now returns `(nil, nil)` for
a stack whose `templates/scaffolds/<stack>/scaffold.json` exists but has no
`templates/packs/<stack>/` — so `specify packs` doesn't fail on a packless target —
while still erroring on a genuinely unknown stack. (The `target add` call site already
swallowed pack errors to stderr; `specify packs` returned them.) The `stack` reaching
`ProjectPacks` is pre-validated by `LoadManifest`, so this loses no typo safety.

**Gotcha proven during the e2e:** `target add` seeds the `root/` example into
`features/` **only on the first target** in a repo (subsequent adds skip it). Adding
swift-package THEN swift-cli to one repo leaves the 2nd stack's scenarios *dangling*
(bindings present, scenario undeclared) — not a scaffold bug; each stack verifies green
in its own fresh repo. So test each library stack in a SEPARATE throwaway repo.

See also [[dev-workflow]], [[web-scaffold]] (the composition model + the dotfile
`go:embed all:` / repo-`.gitignore` drop trap, which applies to `.swift-format`/
`.gitignore` here too).
