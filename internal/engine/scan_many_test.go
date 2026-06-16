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
