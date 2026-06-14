# Apple scaffold — design

**Status:** stack baseline set by Mark (2026-06-14); **Slice 1 shipped** (the
headless Core harness). Grounded in two real repos: `gourmand` (a Tuist app —
the convert/scaffold target) and `apple-platform-tools` (a raw SwiftPM package —
the binding reference), plus the `mac-dev-skills` AppKit plugin (the agent pack).

> An `apple` target is a native Apple **app** (`kind: app`). Its spec-provable
> behaviour lives in a **headless SwiftPM `Core` package** that builds and tests
> with `swift test` alone; the Tuist-generated app surface (AppKit / UIKit) sits
> on top. `specify verify` runs the Core — never Tuist, an Xcode project, a
> simulator, or signing — so it stays green-on-arrival with only the Xcode
> toolchain.

## The stack (baseline)

| concern | choice |
| --- | --- |
| language · concurrency · state | **Swift** · **Swift Concurrency** · **Swift Observation** (`@Observable`) |
| views | **UIKit** (iOS/iPadOS/tvOS/visionOS) · **AppKit** (macOS) · **SwiftUI** (watchOS) |
| database | **SwiftData** |
| networking · OpenAPI | **URLSession** · **Swift OpenAPI Generator** |
| push | **APNs** |
| tests · format/lint | **Swift Testing** · **swift-format** |
| package · project mgr | **Swift Package Manager** · **Tuist** |
| IDE MCP · distribution | **Xcode MCP** · **TestFlight · App Store · Homebrew · web** |

`gourmand`'s practical implementation overrides this baseline wherever they
differ. Note the deliberate **UIKit/AppKit (not SwiftUI)** choice for the primary
platforms — it is why the agent pack is the AppKit-centric `mac-dev-skills`.

## Why the verify target is the headless Core

In both reference repos all spec-provable behaviour is a pure, UI-free package
(`gourmand`'s `Core` = `GourmandCore` + `GourmandPersistence`; `apple-platform-tools`
is itself a multi-target package). Pointing `verify` at that package sidesteps the
entire "can't run Xcode/Tuist/simulator/signing in CI" problem: `swift test` is
headless, deterministic, and needs no project generation. The Tuist app is
scaffolded as the buildable product, but its AppKit-wiring tests are a Mac-only
secondary, never the gate.

## The binding (canonical: the `.scenario()` trait)

Decided 2026-06-14: the scaffold emits — and the engine treats as canonical — the
Swift Testing **trait** form, matching `apple-platform-tools`:

```swift
@Suite(.spec("story.todo.manage"))
struct TodoManageTests {
    @Test(.scenario("scenario.todo.manage.toggle"))
    func `toggling a to-do flips its completion`() { #expect(…) }
}
```

`SpecTraits.swift` defines the `.spec`/`.scenario` factories. The engine already
joins this with **zero** changes: `internal/engine/verify.go:swiftBindRe` reads
`@Test(.scenario("…")) func \`name\`` from source (the scenario id lives in the
trait, the identity is the raw-identifier function name), and
`internal/reports/events.go:ParseSwiftEvents` reads the test outcomes from the
event-stream NDJSON, where the `displayName` equals that same function name.
`gourmand`'s legacy `@Test("[scenario.id] …")` display-name form is **not**
canonical — converting gourmand migrates its tests to the trait (mechanical).

## Composition (base → variants → features), built in slices

### Slice 1 — headless Core harness ✅

`specify target add <name> --stack apple [--dir apps/<name>]` renders into the
member dir:

```
apps/<name>/
  mise.toml          test (writes the event stream) · build · fmt · lint
  .swift-format      lineLength 100, 4-space, lineBreakBeforeEachArgument
  .gitignore         .build, .swiftpm, the report, Tuist Derived/, *.xcodeproj…
  Core/
    Package.swift    name {{pascal .Name}}, target {{pascal .Name}}Core …
    Sources/Core/    Todo.swift (pure domain) · TodoList.swift (@Observable)
    Tests/CoreTests/ SpecTraits.swift (static) · TodoTests.swift (bound)
```

…plus the seeded example story `features/0001-todo/stories/todo.manage.md` (two
scenarios, both green) into the project root. Target wiring:

```json
"command": "cd {{.Dir}} && mise run test",
"format":  "swift",
"report":  "{{.Dir}}/Core/test.swift-events.ndjson",
"source":  "{{.Dir}}/Core",
"bindings": "scoped"
```

The mise `test` task runs
`swift test --package-path Core --event-stream-output-path test.swift-events.ndjson --event-stream-version 0`
— the report lands at `Core/…` because `--event-stream-output-path` resolves
relative to `--package-path`.

**Dynamic module name over a static dir.** The renderer substitutes file
*contents* (and strips `.tmpl`), but **not directory names**. So the package dir
is the static `Sources/Core` / `Tests/CoreTests`, and `Package.swift` names the
module `{{pascal .Name}}Core` via an explicit `path:` — giving `GourmandCore`-style
names without a templated directory.

**`.tmpl` is for substitution only.** Unlike go-service (whose base `files/` is a
Go package the speckit suite compiles), speckit's CI never compiles Swift. A
Swift file is `.tmpl` iff it contains a template var (`Package.swift.tmpl`,
`TodoTests.swift.tmpl` import the module name); `SpecTraits.swift`, `Todo.swift`,
`TodoList.swift` are plain `.swift`.

### Slice 2 — Tuist app surface ✅

`Project.swift` (one macOS AppKit app target + a unit-test target,
`CODE_SIGNING_ALLOWED=NO`, `bundleId: com.example.<name>`), `macOS/Info.plist`,
`macOS/Sources/App` (the programmatic `@main` `AppDelegate` + a
`MainWindowController` that reads the Core's `@Observable` `TodoList`),
`macOS/Tests/AppSmokeTests.swift`, and mise `generate`/`build`/`launch:macos`/
`test:app` (Tuist pinned in `[tools]`). Verify still targets Core; iOS/UIKit is a
documented mirror, not over-built (gourmand hasn't built it yet either). The
generated `.xcodeproj`/`.xcworkspace`/`Derived/` are gitignored — source of truth
is `Project.swift`. Proven on macOS: `mise run build` (BUILD SUCCEEDED) +
`mise run test:app` (passed) + `verify` (Core, green) + `swift format lint --strict`.

What it cost to find:
- **Tuist anchors on the repo root** — the closest dir with a `.git` or `Tuist/`.
  `generate` fails outside a git repo, so the app surface assumes a real SpecKit
  project (every one is a git repo; gourmand is).
- **The Core needs a `.library` product.** `swift test` joins the test target to the
  source target directly, so Slice 1 shipped no `products:`. The Tuist app consumes
  `.package(product: "<Name>Core")`, which requires the library product — added to
  `Package.swift.tmpl` (forward-compatible; `swift test` ignores it).
- **Import order is name-dependent.** `swift-format`'s `OrderedImports` sorts
  lexicographically, but `import AppKit` vs `import <Name>Core` flips with the name
  (`AppCore` < `AppKit` < `GourmandCore`) — no fixed template order is clean for all
  names. Fix: a **silent phase-1 `swift format --in-place`** pass over the rendered
  member normalizes it. (Run `swift format` directly, not `mise run fmt`, so the
  format step doesn't drag a Tuist install into `target add`.)
- **`verify` doesn't pull Tuist.** The `test` task is `swift test` only; pinning
  Tuist in `[tools]` doesn't make `mise run test` install it (a fresh machine only
  fetches Tuist on the first app task — `generate`/`build`).

### Slice 3 — `--with` features 🟡 (go-service-shaped)

**`swiftdata` ✅** — `--with swiftdata` adds a `<Name>Persistence` source target (a
SwiftData `@Model` record + a `SwiftDataTodoStore` mapping the domain `Todo`),
**strict-memory-safety off** (SwiftData's macros sit outside the proof boundary),
and a bound persistence test that verifies headlessly against a temp-file (reopened)
ModelContainer. Proven: `--with swiftdata` → `verify` **3 passed · 1 locked** + app
build + lint clean; base render unaffected. Two patterns this established for every
later feature:

- **The `Package.swift` composition seam.** Features can't share a file (they render
  in nondeterministic map order), but SPM deps/targets are declarative in the
  manifest. So the **base `Package.swift.tmpl` is the seam**: each feature adds its
  target/product/deps behind `{{ "{{" }}if .Features.<name>}}` and ships only additive source
  files. (go-service dodged this with `go get` scripts; SPM has no equivalent, so the
  seam — the web `providers.tsx` pattern — is the right tool.) The example story is a
  seam too: `--with swiftdata` adds a third scenario through the same gate.
- **One test target, always.** `swift test --event-stream-output-path` has each test
  *target*'s process truncate-write the same file, so **multiple test targets clobber
  one another's events** and the engine sees only one. A feature's tests therefore
  land in the single `CoreTests` target (which gains a conditional dependency on the
  feature's source target), never a target of their own. This also drove extracting
  the `.spec`/`.scenario` traits into a shared **`TestSupport`** library target
  (apple-platform-tools' pattern) so every test target imports them.

**`openapi` ✅** — `--with openapi` adds a contract-first API client via the **Swift
OpenAPI Generator** build-tool plugin: an `<Name>API` target (the plugin generates
`Client`/`Types` from `Sources/API/openapi.yaml` at build time, kept internal), a
public `TodoAPIClient` facade that maps the wire schema to the domain `Todo`, and a
bound test that drives the generated client through a **fake `ClientTransport`** —
the analog of go-service's httptest server, so it verifies offline. Proven: `--with
openapi` → `verify` **3 passed**, and `--with swiftdata --with openapi` → **4
passed** + app build, base unaffected. Specifics:
- The seam adds the `swift-openapi-{generator,runtime,urlsession}` package deps —
  **after `products`, before `targets`** (SwiftPM enforces that argument order; got
  it wrong once). The first `verify`/build resolves them (network) and runs codegen
  (~35s cold); after that it's cached.
- The generated code is `internal`, so the public facade is the only surface the app
  and tests touch — no `accessModifier: public` needed.
- Changing the contract may need a clean build to bust the plugin's codegen cache
  (a fresh scaffold always codegens clean).

**Remaining ⬜:** `push` (APNs registration) · `dist` (TestFlight / App Store /
Homebrew release + signing/notarization advisory, from `appkit-packaging`).

### Slice 4 — the apple stack pack ⬜

Project `mac-dev-skills`' `appkit` plugin into `templates/{skills,rules,agents,
commands}`: the `appkit-dev` agent, the design/grounding rules (sdk-api/sdk-search
verification, accessibility identifiers, semantic colors/typography), and the
**Xcode MCP** wiring (a new harness dimension no prior stack has).

## Product kind & sibling stacks

`apple` → `kind: app`. `swift-package` / `swift-cli` (apple-platform-tools' shape:
a flat package/CLI, `kind: library`) reuse Slice 1's Core templates at the repo
root and fall out nearly free — factored after the app slices.

## Method note

Prototype-first / resolve-by-running, on a Mac: build the real Core green in a
throwaway dir (`swift test` + the event stream + `swift format lint --strict`),
then templatize, then prove green-on-arrival via a fresh
`specify target add … --stack apple` → `specify verify`. speckit's own Linux/Go
CI validates the rendered **structure** (`internal/scaffold/apple_test.go`), not
the Swift build.
