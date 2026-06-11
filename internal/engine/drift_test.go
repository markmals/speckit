package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/markmals/speckit/internal/specmodel"
)

const itemSpec = "---\nid: domain.item\nkind: domain\n---\n# Item\nA thing.\n"

func writeSpecFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func contains(ids []specmodel.SpecID, want string) bool {
	for _, id := range ids {
		if string(id) == want {
			return true
		}
	}
	return false
}

// SPEC: story.engine.lock (scenario.engine.lock.writes-on-green)
// SPEC: story.engine.drift (scenario.engine.drift.never-verified-missing, .edited-spec-red, .reverify-clears, .ignores-mtime)
func TestLockAndDrift(t *testing.T) {
	root := t.TempDir()
	specPath := writeSpecFile(t, root, "specs/models/item.md", itemSpec)

	// never verified -> missing
	r, err := Drift(root, "web")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(r.Missing, "domain.item") {
		t.Fatalf("expected missing before lock, got %+v", r)
	}

	// lock -> clean
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}
	if r, _ = Drift(root, "web"); !contains(r.Clean, "domain.item") || r.HasDrift() {
		t.Fatalf("expected clean after lock, got %+v", r)
	}

	// mtime change only -> still clean (D7: content hash, not mtime)
	future := time.Now().Add(48 * time.Hour)
	if err := os.Chtimes(specPath, future, future); err != nil {
		t.Fatal(err)
	}
	if r, _ = Drift(root, "web"); !contains(r.Clean, "domain.item") {
		t.Error("an mtime-only change must not cause drift (D7)")
	}

	// content edit -> drifted
	writeSpecFile(t, root, "specs/models/item.md", itemSpec+"\nedited.\n")
	if r, _ = Drift(root, "web"); !contains(r.Drifted, "domain.item") || !r.HasDrift() {
		t.Fatalf("expected drift after edit, got %+v", r)
	}

	// re-lock -> clean again
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}
	if r, _ = Drift(root, "web"); !contains(r.Clean, "domain.item") {
		t.Errorf("expected clean after re-lock, got %+v", r)
	}
}
