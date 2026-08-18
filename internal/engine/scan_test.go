package engine

import (
	"os"
	"testing"
)

// TestScanForkSpecsClean scans the fork's own spec library — the project is its
// own first user, so it must scan clean.
//
// SPEC: story.engine.scan (scenario.engine.scan.clean)
// [scenario.engine.scan.clean]
func TestScanForkSpecsClean(t *testing.T) {
	findings, err := Scan(os.DirFS("../..")) // repo root from internal/engine
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		for _, f := range findings {
			t.Errorf("%s  %s  %s", f.Invariant, f.Path, f.Message)
		}
		t.Fatalf("fork spec library is not clean: %d findings", len(findings))
	}
}
