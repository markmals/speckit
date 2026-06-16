package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestSwiftPackageScaffold renders the embedded swift-package scaffold and asserts the
// flat-library wiring: members land in packages/ (memberDir), the target uses the swift
// event-stream format + scoped bindings, the module is named after the member over a
// static source directory (explicit `path:`, since the renderer substitutes file
// contents but not directory names), and the seeded story's scenario sub-ids match the
// `.scenario(...)` traits in the bound test — the consistency that makes a fresh
// `specify verify` green with the Swift toolchain alone (no Xcode, simulator, signing).
func TestSwiftPackageScaffold(t *testing.T) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/swift-package")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "swift-package" || m.MemberDir != "packages" {
		t.Errorf("stack=%q memberDir=%q, want swift-package / packages", m.Stack, m.MemberDir)
	}
	if m.Target.Format != "swift" || m.Target.Bindings != "scoped" {
		t.Errorf("target format=%q bindings=%q, want swift / scoped", m.Target.Format, m.Target.Bindings)
	}
	// The module is named {{pascal .Name}}, so the name must pascal-case to a valid
	// Swift identifier — the manifest declares the "identifier" rule for the guard.
	if m.NameRule != "identifier" {
		t.Errorf("nameRule=%q, want identifier (the module name is {{pascal .Name}})", m.NameRule)
	}
	// Each package is self-contained (its own Package.swift), not a shared-module member.
	if m.SharedModule {
		t.Error("swift-package must not be sharedModule")
	}

	dir := t.TempDir()
	data := Data{Name: "recipe-kit", Dir: "packages/recipe-kit"}
	if _, err := Render(sub, dir, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"mise.toml", ".swift-format", ".gitignore",
		"Package.swift",
		"Sources/Library/SemanticVersion.swift",
		"Tests/Support/SpecTraits.swift",
		"Tests/LibraryTests/VersionTests.swift",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}

	// The module name is derived from the member (recipe-kit -> RecipeKit) but the
	// directory stays static (Sources/Library), reconciled by an explicit `path:`.
	pkg := string(mustRead(t, filepath.Join(dir, "Package.swift")))
	for _, want := range []string{
		`name: "RecipeKit"`,
		`.library(name: "RecipeKit", targets: ["RecipeKit"])`,
		`.target(name: "RecipeKit", path: "Sources/Library")`,
		`.target(name: "TestSupport", path: "Tests/Support")`,
		`name: "RecipeKitTests"`,
		`path: "Tests/LibraryTests"`,
	} {
		if !strings.Contains(pkg, want) {
			t.Errorf("Package.swift missing %q:\n%s", want, pkg)
		}
	}
	if !strings.Contains(nospace(pkg), `dependencies:["RecipeKit","TestSupport"]`) {
		t.Errorf("test target deps must be the library + TestSupport:\n%s", pkg)
	}

	// The library source is verbatim (no template vars): the type is name-agnostic and
	// the module is named via Package.swift, so it must not interpolate the member name.
	src := string(mustRead(t, filepath.Join(dir, "Sources/Library/SemanticVersion.swift")))
	if !strings.Contains(src, "public struct SemanticVersion") {
		t.Errorf("SemanticVersion.swift missing the public type:\n%s", src)
	}

	// The bound test @testable-imports the dynamic module and binds via the `.scenario()`
	// trait on raw-identifier function names — never an id embedded in the display name.
	test := string(mustRead(t, filepath.Join(dir, "Tests/LibraryTests/VersionTests.swift")))
	for _, want := range []string{
		"@testable import RecipeKit",
		"import TestSupport",
		`@Suite(.spec("story.version.compare"))`,
		"@Test(.scenario(\"scenario.version.compare.parse\"))\n",
		"func `a dotted string parses into its three components`()",
	} {
		if !strings.Contains(test, want) {
			t.Errorf("VersionTests.swift missing %q:\n%s", want, test)
		}
	}

	// The mise test task writes the event-stream report the engine joins.
	// [vars] provides package_path = "." enabling the swift family canonical run.
	mise := string(mustRead(t, filepath.Join(dir, "mise.toml")))
	if !strings.Contains(mise, "--event-stream-output-path test.swift-events.ndjson") {
		t.Errorf("mise.toml test task must write the event stream:\n%s", mise)
	}
	if !strings.Contains(mise, "[vars]") {
		t.Errorf("swift-package member mise.toml missing [vars]:\n%s", mise)
	}
	if !strings.Contains(mise, `package_path = "."`) {
		t.Errorf("swift-package member mise.toml missing package_path = \".\" in [vars]:\n%s", mise)
	}
	if !strings.Contains(mise, "--package-path .") {
		t.Errorf("swift-package member mise.toml test run must contain --package-path .:\n%s", mise)
	}

	// The seeded story (root/) and the test's `.scenario(...)` traits must name the same
	// scenario sub-ids, or a fresh verify would dangle.
	root := t.TempDir()
	if _, err := RenderRoot(sub, root, data); err != nil {
		t.Fatal(err)
	}
	story := string(mustRead(t, filepath.Join(root, "features/0001-version/stories/version.compare.md")))
	for _, scen := range []string{"scenario.version.compare.parse", "scenario.version.compare.order"} {
		if !strings.Contains(story, "<!-- id: "+scen+" -->") {
			t.Errorf("seeded story missing scenario sub-id %q", scen)
		}
		if !strings.Contains(test, `.scenario("`+scen+`")`) {
			t.Errorf("seeded test missing the .scenario trait for %q", scen)
		}
	}

	// The rendered target wiring: verify runs `mise run test` from the member dir, and
	// the report lands at the member root where --event-stream-output-path writes it.
	rt, err := RenderTarget(m, data)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Command != "mise //packages/recipe-kit:test" {
		t.Errorf("target command = %q", rt.Command)
	}
	if rt.Report != "packages/recipe-kit/test.swift-events.ndjson" || rt.Source != "packages/recipe-kit" {
		t.Errorf("target report=%q source=%q", rt.Report, rt.Source)
	}
}
