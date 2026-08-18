package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/reports"
	"github.com/markmals/speckit/internal/specmodel"
)

func decl(ids ...specmodel.SpecID) map[specmodel.SpecID]bool {
	m := map[specmodel.SpecID]bool{}
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// SPEC: story.engine.verify (scenario.engine.verify.source-bound-join)
func TestLeadingCommentBindingGo(t *testing.T) {
	src := "// [scenario.demo.cli.convert]\nfunc TestConvert(t *testing.T) {}\n\nfunc TestHelper(t *testing.T) {}\n"
	bs := bindingsInContent("cmd/x/x_test.go", src)
	if len(bs) != 1 {
		t.Fatalf("expected 1 Go binding (TestHelper is untagged), got %d: %+v", len(bs), bs)
	}
	if bs[0].Scenario != "scenario.demo.cli.convert" || bs[0].Identity != "TestConvert" || bs[0].Line != 1 {
		t.Errorf("Go leading-comment binding wrong: %+v", bs[0])
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.source-bound-join)
func TestLeadingCommentBindingTS(t *testing.T) {
	src := "describe(\"x\", () => {\n    // [scenario.demo.cli.convert]\n    it(\"does the thing\", () => {})\n})\n"
	bs := bindingsInContent("web/x.test.ts", src)
	if len(bs) != 1 {
		t.Fatalf("expected 1 TS binding, got %d: %+v", len(bs), bs)
	}
	if bs[0].Scenario != "scenario.demo.cli.convert" || bs[0].Identity != "does the thing" {
		t.Errorf("TS leading-comment binding wrong: %+v", bs[0])
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.green-writes-lock)
func TestJoinGreen(t *testing.T) {
	v := Join(
		decl("scenario.a", "scenario.b"),
		[]reports.Result{{Name: "test a", Pass: true}, {Name: "test b", Pass: true}},
		[]Binding{{Scenario: "scenario.a", Identity: "test a"}, {Scenario: "scenario.b", Identity: "test b"}},
	)
	if !v.Green() {
		t.Fatalf("expected green, got %+v", v)
	}
	if len(v.Passed) != 2 {
		t.Errorf("passed: %v", v.Passed)
	}
}

func TestJoinFailing(t *testing.T) {
	v := Join(
		decl("scenario.a"),
		[]reports.Result{{Name: "test a", Pass: false}},
		[]Binding{{Scenario: "scenario.a", Identity: "test a"}},
	)
	if v.Green() || len(v.Failed) != 1 {
		t.Errorf("expected one failure, not green: %+v", v)
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.unjoinable-scenario-fails)
// [scenario.engine.verify.unjoinable-scenario-fails]
func TestJoinUnjoinable(t *testing.T) {
	v := Join(
		decl("scenario.a", "scenario.b"),
		[]reports.Result{{Name: "test a", Pass: true}},
		[]Binding{{Scenario: "scenario.a", Identity: "test a"}},
	)
	if v.Green() {
		t.Error("a declared scenario with no test must fail (D12)")
	}
	if len(v.Unjoinable) != 1 || v.Unjoinable[0] != "scenario.b" {
		t.Errorf("unjoinable: %v", v.Unjoinable)
	}
	if contains(v.Passed, "scenario.b") {
		t.Errorf("an unjoinable scenario must not be reported as passing: %v", v.Passed)
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.dangling-test-ref)
// [scenario.engine.verify.dangling-test-ref]
func TestJoinDangling(t *testing.T) {
	v := Join(
		decl("scenario.a"),
		[]reports.Result{{Name: "test a", Pass: true}, {Name: "ghost", Pass: true}},
		[]Binding{{Scenario: "scenario.a", Identity: "test a"}, {Scenario: "scenario.ghost", Identity: "ghost"}},
	)
	if v.Green() {
		t.Error("a binding to an undeclared scenario must fail (D12)")
	}
	if len(v.Dangling) != 1 || v.Dangling[0].Scenario != "scenario.ghost" {
		t.Errorf("dangling: %v", v.Dangling)
	}
	if len(v.Unbound) != 0 {
		t.Errorf("a bound test (even to a bad scenario) is not unbound: %v", v.Unbound)
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.unbound-test)
// [scenario.engine.verify.unbound-test]
func TestJoinUnbound(t *testing.T) {
	v := Join(
		decl("scenario.a"),
		[]reports.Result{{Name: "test a", Pass: true}, {Name: "orphan", Pass: true}},
		[]Binding{{Scenario: "scenario.a", Identity: "test a"}},
	)
	if v.Green() {
		t.Error("a test with no scenario binding must fail (D12)")
	}
	if len(v.Unbound) != 1 || v.Unbound[0].Name != "orphan" {
		t.Errorf("unbound: %v", v.Unbound)
	}
}

func TestScanBindings(t *testing.T) {
	dir := t.TempDir()
	writeSpecFile(t, dir, "apple/Tests/T.swift",
		"@Test(.scenario(\"scenario.todo.toggle.complete\"))\nfunc `toggling an active todo completes it`() {}\n")
	writeSpecFile(t, dir, "web/test/t.test.ts",
		"it(\"[scenario.todo.toggle.reactivate] reactivates a completed todo\", () => {})\n")
	// a single Swift test pinning two scenarios via multiple .scenario traits —
	// both bind to the one function name (the join identity).
	writeSpecFile(t, dir, "apple/Tests/Multi.swift",
		"@Test(\n    .scenario(\"scenario.todo.toggle.guard\"),\n    .scenario(\"scenario.todo.toggle.empty\")\n)\nfunc `an empty-label toggle is rejected`() throws {}\n")

	bindings, err := ScanBindings(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[specmodel.SpecID]string{}
	for _, b := range bindings {
		got[b.Scenario] = b.Identity
	}
	if got["scenario.todo.toggle.complete"] != "toggling an active todo completes it" {
		t.Errorf("swift binding: %v", got)
	}
	if got["scenario.todo.toggle.reactivate"] != "[scenario.todo.toggle.reactivate] reactivates a completed todo" {
		t.Errorf("vitest binding: %v", got)
	}
	if got["scenario.todo.toggle.guard"] != "an empty-label toggle is rejected" ||
		got["scenario.todo.toggle.empty"] != "an empty-label toggle is rejected" {
		t.Errorf("multi-trait swift: both scenarios must bind the one func name: %v", got)
	}
}

// TestBindingFileAndLine checks bindings carry their source file and 1-based
// line, so CI annotations can point at the exact test.
func TestBindingFileAndLine(t *testing.T) {
	dir := t.TempDir()
	writeSpecFile(t, dir, "web/a.test.ts",
		"import { it } from \"vitest\";\n\nit(\"[scenario.x.one] one\", () => {});\nit(\"[scenario.x.two] two\", () => {});\n")
	bs, err := ScanBindings(dir)
	if err != nil {
		t.Fatal(err)
	}
	byScen := map[specmodel.SpecID]Binding{}
	for _, b := range bs {
		byScen[b.Scenario] = b
	}
	if one := byScen["scenario.x.one"]; one.Line != 3 || !strings.HasSuffix(one.File, "web/a.test.ts") {
		t.Errorf("one: file=%q line=%d, want suffix web/a.test.ts at line 3", one.File, one.Line)
	}
	if two := byScen["scenario.x.two"]; two.Line != 4 {
		t.Errorf("two: line=%d, want 4", two.Line)
	}
}

// TestSpecLocations checks a scenario resolves to its spec file + sub-id line.
func TestSpecLocations(t *testing.T) {
	dir := t.TempDir()
	writeSpecFile(t, dir, "features/0001-x/stories/x.md",
		"---\nid: story.x\nkind: story\n---\n# AC\n\n## Scenario 1: one\n\n<!-- id: scenario.x.one -->\n\n- Given\n")
	locs, err := SpecLocations(dir)
	if err != nil {
		t.Fatal(err)
	}
	loc, ok := locs["scenario.x.one"]
	if !ok {
		t.Fatal("scenario.x.one not located")
	}
	if loc.File != "features/0001-x/stories/x.md" || loc.Line != 9 {
		t.Errorf("loc = %+v, want features/0001-x/stories/x.md line 9", loc)
	}
}

// TestScanBindingsSkipsIgnoredDirs guards the latent gap where a target's
// source dir contains its own node_modules / generated trees: the scan must not
// descend into them (pnpm's symlink-laden node_modules otherwise crashes the
// walk), and must honor the project's .gitignore.
func TestScanBindingsSkipsIgnoredDirs(t *testing.T) {
	dir := t.TempDir()
	writeSpecFile(t, dir, "src/t.test.ts", `it("[scenario.x.real] real", () => {})`+"\n")
	// always-skipped vendored tree
	writeSpecFile(t, dir, "node_modules/dep/d.test.ts", `it("[scenario.x.node] decoy", () => {})`+"\n")
	// a directory the project .gitignore excludes
	writeSpecFile(t, dir, ".gitignore", "dist/\n")
	writeSpecFile(t, dir, "dist/out.test.ts", `it("[scenario.x.dist] decoy", () => {})`+"\n")
	// a directory named like a source file — must not be read as one
	if err := os.MkdirAll(filepath.Join(dir, "src", "weird.ts"), 0o755); err != nil {
		t.Fatal(err)
	}

	bindings, err := ScanBindings(dir)
	if err != nil {
		t.Fatalf("ScanBindings errored on a tree with node_modules/.gitignore: %v", err)
	}
	got := map[specmodel.SpecID]bool{}
	for _, b := range bindings {
		got[b.Scenario] = true
	}
	if !got["scenario.x.real"] {
		t.Errorf("real binding missing: %v", got)
	}
	if got["scenario.x.node"] {
		t.Error("binding under node_modules was scanned (should be skipped)")
	}
	if got["scenario.x.dist"] {
		t.Error("binding under a .gitignore'd dir was scanned (should be skipped)")
	}
}
