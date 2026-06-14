package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestAppleScaffold renders the embedded apple scaffold and asserts the Swift
// wiring: members land in apps/ (memberDir), the target uses the swift event-stream
// format + scoped bindings, the headless Core package renders with a dynamic module
// name over a static directory (explicit `path:`, since the renderer substitutes file
// contents but not directory names), and the seeded story's scenario sub-ids match the
// `.scenario(...)` traits in the bound test — the consistency that makes a fresh
// `specify verify` green without Tuist, an Xcode project, or signing.
func TestAppleScaffold(t *testing.T) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/apple")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "apple" || m.MemberDir != "apps" {
		t.Errorf("stack=%q memberDir=%q, want apple / apps", m.Stack, m.MemberDir)
	}
	if m.Target.Format != "swift" || m.Target.Bindings != "scoped" {
		t.Errorf("target format=%q bindings=%q, want swift / scoped", m.Target.Format, m.Target.Bindings)
	}
	// Unlike go-service, an apple member is a self-contained SwiftPM package, not a
	// shared-module cmd/ — each app owns its Core.
	if m.SharedModule {
		t.Error("apple must not be sharedModule (each app owns its own Core package)")
	}

	dir := t.TempDir()
	data := Data{Name: "gourmand", Dir: "apps/gourmand"}
	if _, err := Render(sub, dir, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"mise.toml", ".swift-format", ".gitignore",
		"Core/Package.swift",
		"Core/Sources/Core/Todo.swift", "Core/Sources/Core/TodoList.swift",
		"Core/Tests/CoreTests/SpecTraits.swift", "Core/Tests/CoreTests/TodoTests.swift",
		// Slice 2 — the Tuist app surface.
		"Project.swift", "macOS/Info.plist",
		"macOS/Sources/App/AppDelegate.swift", "macOS/Sources/App/MainWindowController.swift",
		"macOS/Tests/AppSmokeTests.swift",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}

	// The module name is derived from the app name (Gourmand -> GourmandCore) but the
	// directory stays static (Sources/Core), reconciled by an explicit `path:`.
	pkg, _ := os.ReadFile(filepath.Join(dir, "Core/Package.swift"))
	for _, want := range []string{
		`name: "Gourmand"`,
		// the library product the Tuist app consumes (Slice 2 — swift test alone doesn't need it).
		`.library(name: "GourmandCore", targets: ["GourmandCore"])`,
		`.target(name: "GourmandCore", path: "Sources/Core")`,
		`name: "GourmandCoreTests"`,
		`path: "Tests/CoreTests"`,
	} {
		if !strings.Contains(string(pkg), want) {
			t.Errorf("Package.swift missing %q:\n%s", want, pkg)
		}
	}

	// Slice 2 — the Tuist app project consumes the Core package's product, builds a
	// macOS AppKit app target + a unit-test target, and ships with code-signing off.
	proj, _ := os.ReadFile(filepath.Join(dir, "Project.swift"))
	for _, want := range []string{
		`name: "Gourmand"`,
		`.package(path: "Core")`,
		`"CODE_SIGNING_ALLOWED": "NO"`,
		`bundleId: "com.example.gourmand"`,
		`product: .app`,
		`.package(product: "GourmandCore")`,
		`sources: ["macOS/Sources/**"]`,
		`name: "GourmandTests"`,
		`product: .unitTests`,
	} {
		if !strings.Contains(string(proj), want) {
			t.Errorf("Project.swift missing %q:\n%s", want, proj)
		}
	}

	// The AppKit view layer links the Core (its @Observable model); the entry point is
	// the programmatic @main delegate; the app test @testable-imports the app module.
	win, _ := os.ReadFile(filepath.Join(dir, "macOS/Sources/App/MainWindowController.swift"))
	if !strings.Contains(string(win), "import GourmandCore") || !strings.Contains(string(win), `window.title = "Gourmand"`) {
		t.Errorf("MainWindowController.swift must import the Core and title the window:\n%s", win)
	}
	del, _ := os.ReadFile(filepath.Join(dir, "macOS/Sources/App/AppDelegate.swift"))
	if !strings.Contains(string(del), "@main") || !strings.Contains(string(del), "static func main()") {
		t.Errorf("AppDelegate.swift must be the programmatic @main entry:\n%s", del)
	}
	app, _ := os.ReadFile(filepath.Join(dir, "macOS/Tests/AppSmokeTests.swift"))
	if !strings.Contains(string(app), "@testable import Gourmand") {
		t.Errorf("AppSmokeTests.swift must @testable-import the app module:\n%s", app)
	}

	// The mise config pins Tuist and exposes the app loop (generate/build/test:app),
	// with the scheme named after the app.
	mise, _ := os.ReadFile(filepath.Join(dir, "mise.toml"))
	for _, want := range []string{`tuist = "`, "[tasks.build]", "tuist xcodebuild build -scheme Gourmand", `[tasks."test:app"]`} {
		if !strings.Contains(string(mise), want) {
			t.Errorf("mise.toml missing %q:\n%s", want, mise)
		}
	}

	// The bound test imports the dynamic module and binds via the `.scenario()` trait
	// (the engine's swiftBindRe form) on raw-identifier function names — never the
	// scenario id embedded in the test's display string.
	test, _ := os.ReadFile(filepath.Join(dir, "Core/Tests/CoreTests/TodoTests.swift"))
	for _, want := range []string{
		"@testable import GourmandCore",
		`@Suite(.spec("story.todo.manage"))`,
		"@Test(.scenario(\"scenario.todo.manage.toggle\"))\n",
		"func `toggling a to-do flips its completion`()",
	} {
		if !strings.Contains(string(test), want) {
			t.Errorf("TodoTests.swift missing %q:\n%s", want, test)
		}
	}

	// SpecTraits.swift is stack-invariant (no template substitution) and defines the
	// `.spec`/`.scenario` factories the binding form relies on.
	traits, _ := os.ReadFile(filepath.Join(dir, "Core/Tests/CoreTests/SpecTraits.swift"))
	for _, want := range []string{"struct ScenarioTrait: TestTrait", "static func scenario(_ id: String)"} {
		if !strings.Contains(string(traits), want) {
			t.Errorf("SpecTraits.swift missing %q:\n%s", want, traits)
		}
	}

	// The seeded story (root/) and the test's `.scenario(...)` traits must name the
	// same scenario sub-ids, or a fresh verify would dangle.
	root := t.TempDir()
	if _, err := RenderRoot(sub, root, data); err != nil {
		t.Fatal(err)
	}
	story, _ := os.ReadFile(filepath.Join(root, "features/0001-todo/stories/todo.manage.md"))
	for _, scen := range []string{"scenario.todo.manage.toggle", "scenario.todo.manage.add"} {
		if !strings.Contains(string(story), "<!-- id: "+scen+" -->") {
			t.Errorf("seeded story missing scenario sub-id %q", scen)
		}
		if !strings.Contains(string(test), `.scenario("`+scen+`")`) {
			t.Errorf("seeded test missing the .scenario trait for %q", scen)
		}
	}

	// The rendered target wiring: verify runs `mise run test` from the member dir, and
	// the report lands under Core/ where --event-stream-output-path writes it.
	rt, err := RenderTarget(m, data)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Command != "cd apps/gourmand && mise run test" {
		t.Errorf("target command = %q", rt.Command)
	}
	if rt.Report != "apps/gourmand/Core/test.swift-events.ndjson" || rt.Source != "apps/gourmand/Core" {
		t.Errorf("target report=%q source=%q", rt.Report, rt.Source)
	}
}
