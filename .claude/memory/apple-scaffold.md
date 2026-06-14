# Apple scaffold — working knowledge

The `apple` stack scaffold (`internal/coreassets/templates/scaffolds/apple/`).
Design: [`docs/design/scaffolds/apple.md`](../../docs/design/scaffolds/apple.md).
Baseline + reference repos: gourmand (Tuist app, the convert target),
apple-platform-tools (raw SwiftPM, the binding reference), mac-dev-skills (the
AppKit agent pack). Built in slices; **Slice 1 (headless Core) + Slice 2 (Tuist app surface) shipped
2026-06-14**.

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

- **Structure-only Go test**: `internal/scaffold/apple_test.go` asserts the rendered
  tree + the dynamic-module/static-dir wiring + story↔`.scenario` id agreement. The
  Swift build/verify is proven by a Mac e2e (`target add` → `verify`), never by
  speckit's CI.

See also [[dev-workflow]], [[web-scaffold]] (the composition model + the dotfile
`go:embed all:` / repo-`.gitignore` drop trap, which applies to `.swift-format`/
`.gitignore` here too).
