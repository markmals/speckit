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

**`dist` ✅ (as a deploy kind, not a `--with` feature)** — release/distribution is a
poor fit for `--with` (scaffold features render only into the member dir, but release
workflows belong at the repo-root `.github/`), and SpecKit already has the right home:
the **deploy-kind** subsystem. So `dist` shipped as the **`app-store-connect`** deploy
kind: `specify deploy add app-store-connect <target>` drops a macOS GitHub Actions
workflow that archives with `xcodebuild` and uploads to TestFlight / App Store with the
[`asc`](https://github.com/rorkai/App-Store-Connect-CLI) CLI (`brew install asc`;
`asc auth login --bypass-keychain` + `asc builds upload --archive-path …`) on a `v*` tag,
and records the App Store Connect API key + signing-cert secrets as op:// refs (plus the
non-secret `ASC_APP_ID` repo variable). Covered by `TestRenderDeployAllKinds` +
actionlint. Developer ID (notarized .dmg) and Homebrew are natural future deploy kinds.

**Deferred — `push` (APNs):** mostly config (registration + `.entitlements` +
`Project.swift` capability seam), only a thin token-format kernel is offline-testable,
and actual push needs signing the scaffold disables. gourmand (a local app) has no
server, so it was deferred rather than built as an unused feature.

### Slice 4 — the apple stack pack ✅

The `apple` **pack** (`templates/packs/apple/`) carries the AppKit slice generated
from [`mac-dev-skills`](https://github.com/markmals/mac-dev-skills), alongside the
existing `ios-development` + `ios-simulator-control` skills. The generated slice
includes **appkit-design** (the flagship — `sdk-api`/`sdk-search` grounding,
accessibility identifiers, semantic colors/typography, content-derived window sizing,
Liquid Glass), **apple-hig** (the complete offline HIG corpus), **appkit-private-apis**,
and **appkit-app-inspector**. Skills project to `.claude/skills/` (or `.agents/` /
`.github/`) by `specify target add --stack apple` / `specify packs`; the per-stack
`appkit-dev` agent projects to `.claude/agents/` for adapters with an `AgentsDir()`.

What this slice settled:
- **Whole-directory projection.** Pack skills are copied with their `references/` trees,
  not just `SKILL.md`; `TestProjectPacks` asserts deep references, the offline HIG corpus,
  and the per-stack agent.
- **Generated, not hand-edited.** `scripts/generate-apple-pack.sh` regenerates the AppKit
  slice from `mac-dev-skills`; `scripts/generate-apple-pack.sh --check` runs in CI against
  an external checkout to detect drift. The upstream chain is
  `apple-platform-tools` CLI contracts → `mac-dev-skills` skill references → SpecKit's
  embedded apple pack copy.
- **Per-stack agents, no MCP projection.** A reserved `agents/` pack subdir projects
  stack agents into the adapter's `AgentsDir()` (`.claude/agents/` today); Codex/generic/
  Copilot skip stack agents because they have no agent directory. Xcode MCP stays
  per-machine (user/local config), documented in-skill, **not** projected — by SpecKit's
  deliberate design (per-machine tools aren't committed).

A pack projects only when `.speckit/specs.json` records an `agent` — `specify init
--integration <agent>` seeds it (so `target add`/`packs` auto-project the pack;
`AddTarget` preserves it). This fixed a gap that had silently disabled pack projection
for **every** stack.

## Product kind & sibling stacks

`apple` → `kind: app`. **`swift-package` / `swift-cli` (shipped)** are sibling
library stacks (apple-platform-tools' shape) that reuse Slice 1's headless harness
— the shared `TestSupport` target with the `.spec`/`.scenario` traits, the `swift`
event-stream test task, and scoped bindings — with **zero engine changes**. Both
place members in `packages/` (`memberDir`); the verify target is the package itself
(`swift test`), so they're green-on-arrival with only the Swift toolchain (no Tuist,
Xcode, simulator, or signing).

- **`swift-package`** — a flat reusable library. The module is named after the
  member (`{{pascal .Name}}`) over a static `Sources/Library` directory via an
  explicit `path:` (the renderer substitutes file contents, not directory names).
  The seeded example is a pure `SemanticVersion` (parse + numeric ordering) bound to
  `story.version.compare` — deliberately distinct from `apple`'s `todo` so the two
  never collide on a shared repo's `features/`.
- **`swift-cli`** — a library Core plus a thin **swift-argument-parser** executable
  shell: `{{pascal .Name}}Core` (over `Sources/Core`) holds the provable logic, and a
  `{{kebab .Name}}` `executableTarget` (over `Sources/CLI`, `@main`/`ParsableCommand`)
  parses arguments and delegates. Keeping the behaviour in the library is what lets
  `swift test` verify it headlessly — the binary is never run during verify. The
  package-level `dependencies` sit after `products` (SwiftPM's required arg order).
  Seeded example: a `greet` command bound to `story.greet.run`.

Neither ships a pack, an MCP wiring, or `--with` features (the example is the proof,
not a feature surface). A packless stack is now first-class: `loadPack` returns no
skills for a real scaffold that ships no pack (so `specify packs` doesn't fail on a
packless target) while still erroring on a genuinely unknown stack name. Proven on a
Mac end to end (`swift build` · `swift test` · `swift format lint --strict` ·
`specify verify` 2 passed · 1 locked; the swift-cli binary runs); render-tested in Go
CI via `internal/scaffold/swift_{package,cli}_test.go`.

## Method note

Prototype-first / resolve-by-running, on a Mac: build the real Core green in a
throwaway dir (`swift test` + the event stream + `swift format lint --strict`),
then templatize, then prove green-on-arrival via a fresh
`specify target add … --stack apple` → `specify verify`. speckit's own Linux/Go
CI validates the rendered **structure** (`internal/scaffold/apple_test.go`), not
the Swift build.
