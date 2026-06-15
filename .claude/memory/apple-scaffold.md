# Apple scaffold — working knowledge

The `apple` stack scaffold (`internal/coreassets/templates/scaffolds/apple/`).
Design: [`docs/design/scaffolds/apple.md`](../../docs/design/scaffolds/apple.md).
Baseline + reference repos: gourmand (Tuist app, the convert target),
apple-platform-tools (raw SwiftPM, the binding reference), mac-dev-skills (the
AppKit agent pack). Built in slices; **Slice 1 (headless Core) + Slice 2 (Tuist app surface) + Slice 3
`swiftdata` + `openapi` shipped 2026-06-14**.

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

See also [[dev-workflow]], [[web-scaffold]] (the composition model + the dotfile
`go:embed all:` / repo-`.gitignore` drop trap, which applies to `.swift-format`/
`.gitignore` here too).
