package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// lockWriterSnapshot walks .speckit/lock/ and returns every file's relative
// path mapped to its exact bytes — a byte-level snapshot of the lock tree.
func lockWriterSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	lockDir := filepath.Join(root, ".speckit", "lock")
	err := filepath.Walk(lockDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(lockDir, p)
		if err != nil {
			return err
		}
		snap[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func lockWriterSnapshotsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Only `specify lock` (via verify-on-green) writes drift state: scan, drift,
// cover, and parity must neither write a new lock shard nor mutate an existing
// one (D7 / L1). The test seeds a lock shard, byte-snapshots the whole
// .speckit/lock/ tree, runs every other command, and asserts the tree is
// byte-identical with no new files.
//
// SPEC: story.engine.lock (scenario.engine.lock.sole-writer)
// [scenario.engine.lock.sole-writer]
func TestLockSoleWriter(t *testing.T) {
	root := setupVerifyProject(t, junitReport(true, true))

	// an existing shard with distinctive bytes — any rewrite would change them
	prior := Shard{SpecHash: "sole-writer-sentinel", Scenarios: map[string]string{"scenario.demo.toggle.a": "pass"}}
	if err := WriteShard(root, "web", "story.demo.toggle", prior); err != nil {
		t.Fatal(err)
	}
	before := lockWriterSnapshot(t, root)
	if len(before) == 0 {
		t.Fatal("fixture must start with a lock shard on disk")
	}

	cfg := VerifyConfig{Format: "junit", Report: "web/report.junit.xml", Source: []string{"web"}}

	if _, err := Scan(os.DirFS(root)); err != nil {
		t.Fatal(err)
	}
	if !lockWriterSnapshotsEqual(before, lockWriterSnapshot(t, root)) {
		t.Fatal("Scan wrote or mutated a lock shard — only `specify lock` may (L1)")
	}

	if _, err := Drift(root, "web"); err != nil {
		t.Fatal(err)
	}
	if !lockWriterSnapshotsEqual(before, lockWriterSnapshot(t, root)) {
		t.Fatal("Drift wrote or mutated a lock shard — only `specify lock` may (L1)")
	}

	if _, err := Cover(root, "story.demo.toggle"); err != nil {
		t.Fatal(err)
	}
	if !lockWriterSnapshotsEqual(before, lockWriterSnapshot(t, root)) {
		t.Fatal("Cover wrote or mutated a lock shard — only `specify lock` may (L1)")
	}

	if _, err := Parity(root, "web", cfg); err != nil {
		t.Fatal(err)
	}
	if !lockWriterSnapshotsEqual(before, lockWriterSnapshot(t, root)) {
		t.Fatal("Parity wrote or mutated a lock shard — only `specify lock` may (L1)")
	}

	// control: the one legitimate writer does change the tree
	if err := Lock(root, "web", "story.demo.toggle"); err != nil {
		t.Fatal(err)
	}
	if lockWriterSnapshotsEqual(before, lockWriterSnapshot(t, root)) {
		t.Fatal("control failed: Lock must rewrite the shard, so the snapshot comparison is inert")
	}
}
