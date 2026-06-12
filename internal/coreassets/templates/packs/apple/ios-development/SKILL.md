---
name: ios-development
description: Use when writing or modifying Apple app code for an iOS/macOS/watchOS target. Covers UIKit (AppKit on macOS, SwiftUI on watchOS) + Observation + SwiftData + Swift Testing idioms, the generated OpenAPI client, and points at Apple's first-party docs. Complementary to `implementing-a-spec` (process) and `ios-simulator-control` (simulator driving).
---

# Apple Development

This skill covers **how to write Apple-target code** in this repo. For the _workflow_ of implementing a spec, see `implementing-a-spec`. For _driving the simulator_ (boot, install, screenshot, tap), see `ios-simulator-control`. For _what to build_, see the spec.

**One Swift codebase, the whole Apple family.** The Apple target is a single Swift/SwiftData codebase whose targets cover iOS, iPadOS, macOS, tvOS, watchOS, and visionOS via SwiftPM/Tuist. The UI framework is **per-target**: **UIKit** on iOS · iPadOS · tvOS · visionOS, **AppKit** on macOS, and **SwiftUI** on watchOS. iOS (UIKit) and macOS (AppKit) are the worked examples; every target shares the same view-model and domain layers and diverges only at the view edge (mark those `// SPEC: manual` or `(deviates: …)`). Don't enumerate every target up front — add a target when the product actually ships it.

## Stack at a glance

| Concern                             | Choice                                                                                | First-party docs                                                                                                           |
| ----------------------------------- | ------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| Language                            | Swift (latest stable)                                                                 | [docs.swift.org/swift-book](https://docs.swift.org/swift-book/)                                                            |
| UI (iOS · iPadOS · tvOS · visionOS) | UIKit                                                                                 | [developer.apple.com/documentation/uikit](https://developer.apple.com/documentation/uikit)                                 |
| UI (macOS)                          | AppKit                                                                                | [developer.apple.com/documentation/appkit](https://developer.apple.com/documentation/appkit)                               |
| UI (watchOS)                        | SwiftUI                                                                               | [developer.apple.com/documentation/swiftui](https://developer.apple.com/documentation/swiftui)                             |
| Reactive state                      | `@Observable` macro (Observation framework)                                           | [developer.apple.com/documentation/observation](https://developer.apple.com/documentation/observation)                     |
| Concurrency                         | Swift Concurrency (`async`/`await`, `Task`, `Actor`)                                  | [docs.swift.org concurrency](https://docs.swift.org/swift-book/documentation/the-swift-programming-language/concurrency/)  |
| Tests                               | Swift Testing                                                                         | [developer.apple.com/documentation/testing](https://developer.apple.com/documentation/testing)                             |
| Design language                     | Apple Human Interface Guidelines                                                      | [developer.apple.com/design/human-interface-guidelines](https://developer.apple.com/design/human-interface-guidelines)     |
| Package manager                     | Swift Package Manager                                                                 | [swift.org/package-manager](https://www.swift.org/package-manager/)                                                        |
| Linter / formatter                  | swift-format                                                                          | [github.com/swiftlang/swift-format](https://github.com/swiftlang/swift-format)                                             |
| Project generation                  | Tuist (Swift manifests, no checked-in `.xcodeproj`)                                   | [docs.tuist.dev](https://docs.tuist.dev)                                                                                   |
| On-device database                  | SwiftData                                                                             | [developer.apple.com/documentation/swiftdata](https://developer.apple.com/documentation/swiftdata)                         |
| Networking                          | URLSession                                                                            | [developer.apple.com/documentation/foundation/urlsession](https://developer.apple.com/documentation/foundation/urlsession) |
| API client                          | Swift OpenAPI Generator (typed client over URLSession; do **not** hand-roll requests) | [github.com/apple/swift-openapi-generator](https://github.com/apple/swift-openapi-generator)                               |
| Push notifications                  | APNs (UserNotifications)                                                              | [developer.apple.com/documentation/usernotifications](https://developer.apple.com/documentation/usernotifications)         |
| Auth                                | Clerk (identity / token)                                                              | [clerk.com/docs](https://clerk.com/docs)                                                                                   |

Apple doesn't publish a `/llms.txt` for any of these. Use WebFetch against the canonical doc URLs above when you need to look something up.

## The Client layer depends on the backend mode

How the app reaches its data is set by the project's backend (see `specs/ARCHITECTURE.md` → "Backend modes"):

- **OpenAPI** — a typed client generated by Swift OpenAPI Generator over **URLSession**; never assemble `URLRequest`s or model the wire protocol by hand.
- **Convex** — Convex's first-party Swift client.
- **No API** — no client at all; **SwiftData** is the source of truth.

In the remote modes, **SwiftData** is a local-first cache (not a second backend) and identity flows through **Clerk**, whose token the client attaches. See the target source's `CLAUDE.md` for the wrapper that exposes idiomatic call sites.

## Idioms (read these before writing code)

### `@Observable` for view models, not Combine

The view model is the **primary spec target** and is **UI-framework-agnostic** — the same `@Observable` class drives a UIKit view controller, an AppKit view controller, and a SwiftUI watch view.

```swift
// SPEC: vm.items.list
import Observation

@Observable
@MainActor
final class ItemsListViewModel {
    private(set) var items: [Item] = []
    private(set) var status: Status = .idle

    enum Status: Equatable {
        case idle
        case loading
        case loaded
        case error(String)
    }

    private let client: APIClient

    init(client: APIClient) {
        self.client = client
    }

    func load() async {
        status = .loading
        do {
            items = try await client.listItems()
            status = .loaded
        } catch {
            status = .error(error.localizedDescription)
        }
    }
}
```

`APIClient` here is the thin wrapper over the Swift OpenAPI Generator output — it exposes idiomatic async methods (`listItems()`), not raw generated operation names.

- `@Observable` macro generates the observation plumbing — no manual `@Published` or `ObservableObject`.
- View models are **reference types** (`final class`), `@MainActor`-pinned so UI reads are safe by default.
- Use Combine **only** when interoperating with an existing Combine API. Reach for it last.

### Async/await for everything I/O

No completion handlers, no `DispatchQueue.main.async`. If you need to bridge a completion-handler API, wrap it once with `withCheckedContinuation` and forget about it.

### Dependencies via initializer or property injection

No singletons. The API client and other services are passed into a view controller (or, on watchOS, a SwiftUI view) via its initializer. This is what makes testing tractable — a view model under test gets a mock client. (On watchOS you may use SwiftUI's `Environment` for ambient dependencies; UIKit/AppKit use plain initializer/property injection.)

### Tests at the view-model layer

Bind tests to spec IDs using the Swift Testing custom traits shipped in `SpecTraits.swift`. The spec ID goes in the trait — never in the test description. Use raw-identifier function names for the human-readable description.

```swift
import Testing

@Suite(.spec("vm.items.list"))
struct ItemsListViewModelTests {

    @Test(.scenario("scenario.items.list.empty"))
    func `shows empty state when no items exist`() async {
        let client = MockAPIClient(returning: [])
        let vm = ItemsListViewModel(client: client)

        await vm.load()

        #expect(vm.items.isEmpty)
        #expect(vm.status == .loaded)
    }
}
```

- `@Suite(.spec("<id>"))` — the spec ID, read by the scanner.
- `@Test(.scenario("<sub-id>"))` — the scenario sub-ID from `specs/CONVENTIONS.md`; the engine joins this to the report by test identity.
- Raw-identifier backtick names carry the human description; the dotted ID stays in the trait.
- Use `#expect` and `#require` (Swift Testing macros) instead of XCTest's `XCTAssert*`.
- The report format for this target is `swift` (Swift Testing's event-stream NDJSON); `specify verify <target>` reads it.

### UIKit views are dumb — drive them from `@Observable` via `updateProperties()`

A view controller consumes a view model and renders. No business logic in the controller. On iOS · iPadOS · tvOS · visionOS, read the view model inside **`updateProperties()`**: UIKit automatically tracks the `@Observable` properties you read there and re-invokes the method whenever they change — the UIKit analogue of SwiftUI's `body`. No manual KVO, no diffing subscriptions.

```swift
final class ItemsListViewController: UIViewController {
    private let viewModel: ItemsListViewModel
    private let tableView = UITableView()

    init(viewModel: ItemsListViewModel) {
        self.viewModel = viewModel
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("use init(viewModel:)") }

    override func viewDidLoad() {
        super.viewDidLoad()
        // configure tableView, add subviews, constraints…
        Task { await viewModel.load() }
    }

    // Reads of viewModel.status / viewModel.items here are tracked automatically;
    // UIKit re-runs updateProperties() whenever they change.
    override func updateProperties() {
        super.updateProperties()
        switch viewModel.status {
        case .idle, .loading:
            showLoading()
        case .loaded:
            apply(items: viewModel.items)
        case .error(let message):
            showError(message, retry: { [weak self] in
                Task { await self?.viewModel.load() }
            })
        }
    }
}
```

The reverse pointer lives on the **view model**, not the view controller. Controllers carry `// SPEC: manual` (no cross-target behavioral contract) or `// SPEC: <vm-id> (deviates: <ui reason>)` if there's a UI-level deviation.

### AppKit (macOS): observe an `Observations` async sequence

AppKit has no `updateProperties()`. Drive rendering from an **`Observations`** async sequence — it yields whenever any `@Observable` property read inside the closure changes — consumed on the main actor.

```swift
final class ItemsListViewController: NSViewController {
    private let viewModel: ItemsListViewModel
    private var renderTask: Task<Void, Never>?

    init(viewModel: ItemsListViewModel) {
        self.viewModel = viewModel
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("use init(viewModel:)") }

    override func viewDidLoad() {
        super.viewDidLoad()
        let states = Observations { [viewModel] in viewModel.status }
        renderTask = Task { @MainActor [weak self] in
            for await status in states {
                self?.render(status)
            }
        }
        Task { await viewModel.load() }
    }

    deinit { renderTask?.cancel() }
}
```

### watchOS: SwiftUI, reading the same view model

On watchOS the view is SwiftUI and reads the `@Observable` view model directly — no `updateProperties()`, no `Observations`; SwiftUI tracks it natively. Keep the view dumb the same way (no logic in `body`).

```swift
struct ItemsListView: View {
    @State private var viewModel: ItemsListViewModel

    var body: some View {
        Group {
            switch viewModel.status {
            case .idle, .loading: ProgressView()
            case .loaded: List(viewModel.items) { ItemRow(item: $0) }
            case .error(let message): ItemsErrorState(message: message) {
                Task { await viewModel.load() }
            }
            }
        }
        .task { await viewModel.load() }
    }
}
```

### HIG-driven affordances

When in doubt about a design choice, read the relevant HIG section. Common ones:

- [Navigation](https://developer.apple.com/design/human-interface-guidelines/navigation-and-search)
- [Selection and input](https://developer.apple.com/design/human-interface-guidelines/inputs)
- [Lists and tables](https://developer.apple.com/design/human-interface-guidelines/lists-and-tables)
- [Modality](https://developer.apple.com/design/human-interface-guidelines/modality)

Native affordances (pull-to-refresh via `UIRefreshControl`, swipe actions, share sheets, system back gestures, AppKit toolbars and menus) are encouraged. Mark them `// SPEC: manual` if no cross-target spec applies.

## File layout (within the target's source)

See the target source's `CLAUDE.md` for the canonical layout. Summary:

```
<source>/App/
├── App/                            ← App entry, root scene/window, root controller
├── Features/<Feature>/             ← Feature-scoped: VM, VM tests, view controller(s)
├── Domain/                         ← Plain types and validation (one file per domain.* spec)
├── Client/                         ← generated OpenAPI client wrapper (URLSession transport)
├── Store/                          ← SwiftData models + local-first cache
└── Resources/                      ← Assets, localizable strings, Info.plist
```

## Driving Xcode from the agent (MCP)

Xcode can expose itself to external agents over MCP — building, running, testing, and inspecting the project with structured results instead of scraping `xcodebuild` output. Enable **external agent access** in Xcode, then register the server with Claude Code.

- Apple docs: [Giving external agents access to Xcode](https://developer.apple.com/documentation/xcode/giving-external-agents-access-to-xcode).
- It is **per-machine** — it requires Xcode running with the feature enabled — so it belongs in your **user/local MCP config (`~/.claude/` or `.mcp.local.json`)**, not the committed `.mcp.json`, which must work for everyone (including contributors who don't run Xcode).
- Once connected, prefer it over shelling out for build/test/diagnostics; fall back to the target's own build/test commands when the bridge isn't available. See `ios-simulator-control` for the device side.

## When to invoke a more specific skill

- About to write tests? → `test-driven-development`
- About to claim work is done? → `verification-before-completion`
- Booting the simulator, screenshotting, tapping? → `ios-simulator-control`
- Debugging something unexpected? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`

## Commit

Land focused, atomic commits as the work hits natural boundaries — typically per spec ID, per view-model + its tests, or per cohesive refactor. Use the scoped-commits convention (`specify gate scope`).

iOS-specific notes:

- **Tuist manifest changes go alone.** `Project.swift` / `Workspace.swift` edits belong in their own commit (`chore: add <module> to Tuist project`). Don't bundle with feature code.
- **Don't commit generated Xcode project files.** `.xcodeproj/` is gitignored; regenerated via Tuist on demand.
- **Asset additions are separate.** New images, color sets, or localizable strings belong in their own commit so the diff is reviewable.
