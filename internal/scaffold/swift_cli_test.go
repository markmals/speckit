package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestSwiftCLIScaffold renders the embedded swift-cli scaffold and asserts the
// library-core + thin-executable wiring (apple-platform-tools' shape): members land in
// packages/, the target uses the swift event-stream format + scoped bindings, the
// provable logic is a library target (<Name>Core over a static Sources/Core dir via
// `path:`) while the executable (<name>) is a swift-argument-parser shell over it, and
// the seeded story's scenario sub-ids match the bound test's `.scenario(...)` traits —
// so a fresh `specify verify` is green via `swift test` alone, without running the binary.
func TestSwiftCLIScaffold(t *testing.T) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/swift-cli")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "swift-cli" || m.MemberDir != "packages" {
		t.Errorf("stack=%q memberDir=%q, want swift-cli / packages", m.Stack, m.MemberDir)
	}
	if m.Target.Format != "swift" || m.Target.Bindings != "scoped" {
		t.Errorf("target format=%q bindings=%q, want swift / scoped", m.Target.Format, m.Target.Bindings)
	}
	// {{pascal .Name}}Core is a module name, so the name must pascal-case to a valid
	// Swift identifier — the manifest declares the "identifier" rule for the guard.
	if m.NameRule != "identifier" {
		t.Errorf("nameRule=%q, want identifier (the module name is {{pascal .Name}}Core)", m.NameRule)
	}
	if m.SharedModule {
		t.Error("swift-cli must not be sharedModule")
	}

	dir := t.TempDir()
	data := Data{Name: "greet-tool", Dir: "packages/greet-tool"}
	if _, err := Render(sub, dir, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"mise.toml", ".swift-format", ".gitignore",
		"Package.swift",
		"Sources/Core/Greeter.swift",
		"Sources/CLI/Command.swift",
		"Tests/Support/SpecTraits.swift",
		"Tests/CoreTests/GreeterTests.swift",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}

	// The executable is named for the member (greet-tool); the library module is
	// <Pascal>Core over a static Sources/Core dir via `path:`. swift-argument-parser is
	// an unconditional dependency (the CLI shell needs it).
	pkg := string(mustRead(t, filepath.Join(dir, "Package.swift")))
	for _, want := range []string{
		`name: "GreetTool"`,
		`.executable(name: "greet-tool", targets: ["greet-tool"])`,
		`.library(name: "GreetToolCore", targets: ["GreetToolCore"])`,
		`.package(url: "https://github.com/apple/swift-argument-parser", from: "1.5.0")`,
		`.target(name: "GreetToolCore", path: "Sources/Core")`,
		`.executableTarget(`,
		`path: "Sources/CLI"`,
		`.product(name: "ArgumentParser", package: "swift-argument-parser")`,
		`.target(name: "TestSupport", path: "Tests/Support")`,
		`name: "GreetToolCoreTests"`,
		`path: "Tests/CoreTests"`,
	} {
		if !strings.Contains(pkg, want) {
			t.Errorf("Package.swift missing %q:\n%s", want, pkg)
		}
	}
	// SwiftPM requires `dependencies` to come AFTER `products` (and before `targets`).
	if di, pi := strings.Index(pkg, "dependencies:"), strings.Index(pkg, "products:"); di < pi {
		t.Errorf("Package.swift: `dependencies` must come after `products` (SwiftPM arg order)")
	}

	// The executable is a thin shell: it imports the Core, parses args, and delegates.
	cmd := string(mustRead(t, filepath.Join(dir, "Sources/CLI/Command.swift")))
	for _, want := range []string{
		"import ArgumentParser",
		"import GreetToolCore",
		"@main",
		"struct GreetToolCommand: ParsableCommand",
		`commandName: "greet-tool"`,
		"Greeter.greeting(for: name, shout: shout)",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("Command.swift missing %q:\n%s", want, cmd)
		}
	}

	// The provable logic is verbatim, pure, name-agnostic (the module is named via
	// Package.swift): no I/O, no argument parsing.
	greeter := string(mustRead(t, filepath.Join(dir, "Sources/Core/Greeter.swift")))
	if !strings.Contains(greeter, "public enum Greeter") ||
		!strings.Contains(greeter, "func greeting(for name: String, shout: Bool = false) -> String") {
		t.Errorf("Greeter.swift missing the pure logic:\n%s", greeter)
	}

	// The bound test @testable-imports the Core and binds via `.scenario()`.
	test := string(mustRead(t, filepath.Join(dir, "Tests/CoreTests/GreeterTests.swift")))
	for _, want := range []string{
		"@testable import GreetToolCore",
		"import TestSupport",
		`@Suite(.spec("story.greet.run"))`,
		"@Test(.scenario(\"scenario.greet.run.basic\"))\n",
		"func `greeting a name returns a friendly line`()",
	} {
		if !strings.Contains(test, want) {
			t.Errorf("GreeterTests.swift missing %q:\n%s", want, test)
		}
	}

	// The mise tasks write the event stream the engine joins and expose `run`.
	mise := string(mustRead(t, filepath.Join(dir, "mise.toml")))
	for _, want := range []string{"--event-stream-output-path test.swift-events.ndjson", "swift run greet-tool"} {
		if !strings.Contains(mise, want) {
			t.Errorf("mise.toml missing %q:\n%s", want, mise)
		}
	}

	// The seeded story sub-ids and the test's `.scenario(...)` traits must match.
	root := t.TempDir()
	if _, err := RenderRoot(sub, root, data); err != nil {
		t.Fatal(err)
	}
	story := string(mustRead(t, filepath.Join(root, "features/0001-greet/stories/greet.run.md")))
	for _, scen := range []string{"scenario.greet.run.basic", "scenario.greet.run.shout"} {
		if !strings.Contains(story, "<!-- id: "+scen+" -->") {
			t.Errorf("seeded story missing scenario sub-id %q", scen)
		}
		if !strings.Contains(test, `.scenario("`+scen+`")`) {
			t.Errorf("seeded test missing the .scenario trait for %q", scen)
		}
	}

	// The rendered target wiring.
	rt, err := RenderTarget(m, data)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Command != "mise //packages/greet-tool:test" {
		t.Errorf("target command = %q", rt.Command)
	}
	if rt.Report != "packages/greet-tool/test.swift-events.ndjson" || rt.Source != "packages/greet-tool" {
		t.Errorf("target report=%q source=%q", rt.Report, rt.Source)
	}
}
