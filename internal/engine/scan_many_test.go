package engine

import "testing"

func TestScanBindingsMany(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "cmd/a/a_test.go", "// [scenario.x.one]\nfunc TestOne(t *testing.T) {}\n")
	writeSpecFile(t, root, "internal/b/b_test.go", "// [scenario.x.two]\nfunc TestTwo(t *testing.T) {}\n")

	bs, err := ScanBindingsMany(root, []string{"cmd/a", "internal/b"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, b := range bs {
		got[string(b.Scenario)] = true
	}
	if !got["scenario.x.one"] || !got["scenario.x.two"] {
		t.Fatalf("expected bindings from both roots, got %+v", bs)
	}
}

func TestScanDeviationsMany(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "cmd/a/a.go", "// SPEC: scenario.x.one (deviates: wip)\n")
	writeSpecFile(t, root, "internal/b/b.go", "// SPEC: scenario.x.two (deviates: later)\n")

	devs, err := ScanDeviationsMany(root, []string{"cmd/a", "internal/b"})
	if err != nil {
		t.Fatal(err)
	}
	if devs["scenario.x.one"] != "wip" || devs["scenario.x.two"] != "later" {
		t.Fatalf("expected merged deviations from both roots, got %+v", devs)
	}
}

func TestScanDeviationsManyLastWins(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "cmd/a/a.go", "// SPEC: scenario.x.one (deviates: first)\n")
	writeSpecFile(t, root, "internal/b/b.go", "// SPEC: scenario.x.one (deviates: second)\n")

	devs, err := ScanDeviationsMany(root, []string{"cmd/a", "internal/b"})
	if err != nil {
		t.Fatal(err)
	}
	if devs["scenario.x.one"] != "second" {
		t.Fatalf("later root's reason should win, got %q", devs["scenario.x.one"])
	}
}

func TestScanBindingsManyToleratesMissingRoot(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "cmd/a/a_test.go", "// [scenario.x.one]\nfunc TestOne(t *testing.T) {}\n")
	// "internal/b" is never created — a missing source root is tolerated (no
	// error), matching walkSourceFiles' contract for vendored/absent trees. The
	// present root's bindings still come back; a genuinely wrong path instead
	// surfaces later as unjoinable scenarios, not a silent pass.
	bs, err := ScanBindingsMany(root, []string{"cmd/a", "internal/b"})
	if err != nil {
		t.Fatalf("a missing source root must be tolerated, got %v", err)
	}
	found := false
	for _, b := range bs {
		if b.Scenario == "scenario.x.one" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the present root's binding despite the missing sibling")
	}
}
