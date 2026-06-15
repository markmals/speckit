---
name: appkit-ui-testing
description: Use to drive a macOS AppKit app under XCUITest for visual verification and behavioral checks — querying controls by accessibility identifier, asserting values, exercising sheets/menus/alerts/file panels, capturing screenshots, and running performAccessibilityAudit(). The macOS analog of `ios-simulator-control`.
---

# AppKit UI Testing

XCUITest automation for the macOS app surface (`macOS/Sources/App`). Use it the same way you'd use `ios-simulator-control` on iOS or `chrome-devtools` on the web: tight loops of "make change → launch → query/assert → screenshot → verify". For _writing_ the AppKit views, see `appkit-design`; for the target/build setup, see `appkit-setup`; for the spec-provable domain layer, see `ios-development`.

**XCUITest is code in a UI-test target, not a CLI.** Unlike `idb`, there's no standalone driver — you write an `XCTestCase`, then run the whole suite in one `xcodebuild test` pass and read the result bundle. These tests prove **AppKit-wiring scenarios** that can't be proven in the headless `Core` package; the spec-provable domain still lives in `Core` and runs under `specify verify`. Bind any such scenario with the same `.scenario("…")` id as the Core tests — XCUITest itself is XCTest, so the id lives in a comment, not a trait.

> The whole approach depends on **stable accessibility identifiers**. Every interactive control the `appkit-design` skill produces sets `setAccessibilityIdentifier(_:)` (the scaffold's window already does — `todo-count`), separate from the visible label. Without one, XCUITest can only match by localized label or position — brittle. Add the identifier first.

## Prerequisite: a uiTests target

The scaffold's `Project.swift` ships the app target plus a `…Tests` (unitTests) target — there's **no UI-test target yet**. Add one and regenerate, then run with `mise run -C macOS generate`:

```swift
.target(
    name: "{{App}}UITests",
    destinations: .macOS,
    product: .uiTests,
    bundleId: "com.example.{{app}}.uitests",
    deploymentTargets: .macOS("14.0"),
    sources: ["macOS/UITests/**"],
    dependencies: [.target(name: "{{App}}")]
)
```

Tuist-manifest edits land in their own commit (`chore: add UI-test target`), not bundled with feature code — see `ios-development` → Commit.

## Grounding (before you write AppKit-driving code)

Verify every AppKit symbol and `@available` version with **`sdk-api`** and search canonical query/assertion patterns with **`sdk-search`** — never guess element-type names or method signatures:

```sh
sdk-api check XCUIApplication.performAccessibilityAudit   # exists? macOS availability?
sdk-search "XCUITest query NSPopUpButton value"
```

## Write the test class

XCUITest launches its **own** instance of the app (`XCUIApplication().launch()`); you don't attach to a PID. Pass `launchArguments` / `launchEnvironment` to put the app into a known state (e.g. a temp data dir) so runs are deterministic and don't clobber real data. One assertion group per `test…` method with `continueAfterFailure = false` gives clean per-requirement pass/fail.

```swift
import XCTest

final class UITests: XCTestCase {
    var app: XCUIApplication!

    override func setUpWithError() throws {
        continueAfterFailure = false
        app = XCUIApplication()
        app.launchArguments = ["-uiTest", "1"]   // app reads this to use a temp data dir
        app.launch()
    }

    override func tearDownWithError() throws { app.terminate() }

    func testLaunchShowsMainControls() {
        XCTAssertTrue(app.staticTexts["todo-count"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons["save"].exists)
    }

    // typeText alone does NOT commit an NSTextField binding — see Gotchas
    func testSetUsernameCommits() {
        let field = app.textFields["username"]
        field.click()
        field.typeText("TestUser")
        app.buttons["save"].click()              // end-editing commits the binding
        XCTAssertEqual(field.value as? String, "TestUser")
    }

    // macOS 14+ (the scaffold's deployment target) — no #available gate needed
    func testAccessibilityAudit() throws {
        try app.performAccessibilityAudit()       // contrast, missing labels, hit-region…
    }
}
```

## Querying and asserting

`.value` returns a different type per control — read the right one:

| Control | query / `.value` |
|---|---|
| `NSTextField` (editable) | `app.textFields["id"].value as? String` |
| `NSTextField` (label) | `app.staticTexts["id"].label` |
| `NSPopUpButton` | `app.popUpButtons["id"].value as? String` (selected title) |
| `NSComboBox` | `app.comboBoxes["id"].value as? String` |
| `NSSwitch` / checkbox | `app.switches["id"].value as? Int` (1/0) |
| `NSSlider` | `app.sliders["id"].value as? Double` |
| sidebar / source list | `app.outlines["id"].staticTexts["Row"].click()` |

Common actions: `el.click()`, `.rightClick()`, `.doubleClick()`, `.typeText("…")`, `.waitForExistence(timeout:)`, and `app.typeKey("s", modifierFlags: .command)` for shortcuts. Confirm any unfamiliar query with `sdk-search` rather than guessing.

## Sheets, menus, alerts, file panels

These appear **asynchronously** in the same `app` element tree — always `waitForExistence` after triggering, never `.exists` immediately.

```swift
// Sheet / NSAlert (attached to the parent window; sometimes surfaced as app.dialogs)
app.buttons["delete"].click()
let sheet = app.sheets.firstMatch
XCTAssertTrue(sheet.waitForExistence(timeout: 3))
sheet.buttons["Delete"].click()
XCTAssertFalse(sheet.waitForExistence(timeout: 2))     // dismissed

// Context menu
app.outlines["list"].outlineRows.firstMatch.rightClick()
app.menuItems["Copy"].click()

// Main menu bar
app.menuBars.menuBarItems["File"].click()
app.menuItems["Save As…"].click()

// NSOpenPanel / NSSavePanel — same app tree, no separate process
app.buttons["open"].click()
let panel = app.dialogs.firstMatch
XCTAssertTrue(panel.waitForExistence(timeout: 3))
app.typeKey("g", modifierFlags: [.command, .shift])    // Go-to-folder
app.typeText("/tmp/test.txt\n")
panel.buttons["Open"].click()
```

Alert and menu items match by title — give custom buttons clear titles or set identifiers. Menu items exist in the tree only while the menu is open.

## Run and read results

```bash
mise run -C macOS generate
xcodebuild test \
  -scheme {{App}} \
  -destination 'platform=macOS' \
  -only-testing:{{App}}UITests \
  -resultBundlePath ./TestResults.xcresult
xcrun xcresulttool get test-results summary --path ./TestResults.xcresult
```

App-level unit tests still go through `mise run -C macOS test:app`; XCUITest is the visual/behavioral layer on top. If Xcode's MCP bridge is connected, prefer it over scraping `xcodebuild` output — that bridge is per-machine and lives in your local MCP config, never committed; see `ios-development` → "Driving Xcode from the agent".

## Look at the screenshots

Assertions don't catch clipping, overlap, wrong theming, or controls bleeding past their container — an audit can pass while the window is visually broken. Attach the main window and any post-interaction state, then open the PNGs from the result bundle:

```swift
let shot = XCTAttachment(screenshot: app.windows.firstMatch.screenshot())
shot.name = "01-initial"; shot.lifetime = .keepAlways
add(shot)
```

**Fail the run if any item is `no`:** no unintended scrollbars · no text truncated to `…` that shouldn't be · primary/right-edge/bottom controls fully visible · no overlapping rows · content uses the available width · spacing intentional · theming matches the ask (Light/Dark/Increased Contrast) · Liquid Glass toolbar/sidebar renders and content isn't hidden behind a floating toolbar · focus/error states render if tested. A failed item is a bug — fix before declaring done. Window too small → fix via content-derived sizing (`fittingSize` / Auto Layout), never a magic frame; see `appkit-design`.

## Key gotchas

- **`typeText` does NOT commit a default `NSTextField` binding.** Cocoa commits on **end editing** (focus loss / Return). Typing updates the displayed text but not the model — click another control, press Tab, or press Return after typing, or make the field continuous (`isContinuous = true`). The AppKit twin of an `UpdateSourceTrigger` gotcha.
- **Always `waitForExistence` for async UI** — sheets, menus, popovers, and freshly-loaded panes appear asynchronously; `.exists` checked too early returns `false`.
- **Verify persistence via the data file, not a relaunch** — relaunching inside a test is fragile. Write to a temp dir (`launchArguments`), then read it back with `FileManager` + `JSONDecoder` and assert.
- **Identifiers beat labels** — labels are localized and change; query stable `accessibilityIdentifier`s set with `setAccessibilityIdentifier(_:)`, separate from the label.
- **`NSSecureTextField` value is masked** — assert via app state, not the field value.

## When to invoke a more specific skill

- Writing the AppKit views (semantic colors, typography, Auto Layout, Liquid Glass)? → `appkit-design`
- Setting up the target / build / format? → `appkit-setup`
- About to write tests? → `test-driven-development`
- About to claim work is done? → `verification-before-completion`
- A test fails for a non-obvious reason? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`
