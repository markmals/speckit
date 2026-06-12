package engine

import "testing"

// SPEC: story.engine.cover (scenario.engine.cover.per-platform, .reads-lock, .drifted)
func TestCover(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "specs/models/item.md", itemSpec)
	writeSpecFile(t, root, "specs/models/other.md", "---\nid: domain.other\nkind: domain\n---\n# Other\n")

	// apple locks at v1, then the spec is edited so apple goes stale
	if err := Lock(root, "apple", "domain.item"); err != nil {
		t.Fatal(err)
	}
	writeSpecFile(t, root, "specs/models/item.md", itemSpec+"\nedited.\n")
	// web locks at the current (v2) content
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}
	// linux has lock state (for a different spec) but no shard for domain.item
	if err := Lock(root, "linux", "domain.other"); err != nil {
		t.Fatal(err)
	}

	r, err := Cover(root, "domain.item")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range r.Cells {
		got[c.Platform] = c.State
	}
	want := map[string]string{"apple": "drifted", "linux": "missing", "web": "conforming"}
	for p, w := range want {
		if got[p] != w {
			t.Errorf("platform %s: got %q, want %q", p, got[p], w)
		}
	}
}
