package engine

import (
	"os"
	"path/filepath"
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

// SPEC: story.engine.verify (scenario.engine.verify.green-writes-lock)
func TestJoinGreen(t *testing.T) {
	v := Join(
		decl("scenario.a", "scenario.b"),
		[]reports.Result{{Name: "test a", Pass: true}, {Name: "test b", Pass: true}},
		[]Binding{{"scenario.a", "test a"}, {"scenario.b", "test b"}},
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
		[]Binding{{"scenario.a", "test a"}},
	)
	if v.Green() || len(v.Failed) != 1 {
		t.Errorf("expected one failure, not green: %+v", v)
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.unjoinable-scenario-fails)
func TestJoinUnjoinable(t *testing.T) {
	v := Join(
		decl("scenario.a", "scenario.b"),
		[]reports.Result{{Name: "test a", Pass: true}},
		[]Binding{{"scenario.a", "test a"}},
	)
	if v.Green() {
		t.Error("a declared scenario with no test must fail (D12)")
	}
	if len(v.Unjoinable) != 1 || v.Unjoinable[0] != "scenario.b" {
		t.Errorf("unjoinable: %v", v.Unjoinable)
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.dangling-test-ref)
func TestJoinDangling(t *testing.T) {
	v := Join(
		decl("scenario.a"),
		[]reports.Result{{Name: "test a", Pass: true}, {Name: "ghost", Pass: true}},
		[]Binding{{"scenario.a", "test a"}, {"scenario.ghost", "ghost"}},
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
func TestJoinUnbound(t *testing.T) {
	v := Join(
		decl("scenario.a"),
		[]reports.Result{{Name: "test a", Pass: true}, {Name: "orphan", Pass: true}},
		[]Binding{{"scenario.a", "test a"}},
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
