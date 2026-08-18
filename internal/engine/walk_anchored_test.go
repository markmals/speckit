package engine

import (
	"testing"

	"github.com/markmals/speckit/internal/specmodel"
)

// TestScanBindingsHonorsGitignoreAnchoring guards a bug that silently faked a
// green verify: a root-anchored .gitignore entry (`/specify`, the built binary)
// was being turned into a global directory-name skip, so the walk skipped the
// unrelated source directory cmd/specify and every binding under it vanished
// from the scan. Git would never ignore cmd/specify for `/specify`.
func TestScanBindingsHonorsGitignoreAnchoring(t *testing.T) {
	dir := t.TempDir()
	// `/specify` is the repo-root binary; `build/` is an unanchored generated tree.
	writeSpecFile(t, dir, ".gitignore", "/specify\nbuild/\n")
	writeSpecFile(t, dir, "cmd/specify/main_test.go", "// [scenario.x.anchored]\nfunc TestAnchored(t *testing.T) {}\n")
	writeSpecFile(t, dir, "build/out_test.go", "// [scenario.x.generated]\nfunc TestGenerated(t *testing.T) {}\n")

	bindings, err := ScanBindings(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[specmodel.SpecID]bool{}
	for _, b := range bindings {
		got[b.Scenario] = true
	}
	if !got["scenario.x.anchored"] {
		t.Error("cmd/specify was skipped: a root-anchored /specify entry must not skip a nested directory of the same name")
	}
	if got["scenario.x.generated"] {
		t.Error("build/ was scanned: an unanchored .gitignore directory entry must still be skipped")
	}
}
