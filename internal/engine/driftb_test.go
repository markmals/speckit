package engine

import (
	"os"
	"testing"
	"time"
)

// A spec verified green whose content is then edited must be reported drifted
// (hash mismatch), and the report must carry the non-zero-exit condition.
//
// [scenario.engine.drift.edited-spec-red]
func TestDriftEditedSpecRed(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "specs/models/item.md", itemSpec)
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}
	writeSpecFile(t, root, "specs/models/item.md", itemSpec+"\nedited.\n")

	r, err := Drift(root, "web")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(r.Drifted, "domain.item") {
		t.Fatalf("an edited spec must be reported drifted, got %+v", r)
	}
	if contains(r.Clean, "domain.item") {
		t.Errorf("an edited spec must not also be clean, got %+v", r)
	}
	if !r.HasDrift() {
		t.Error("a drifted spec must make the command exit non-zero (HasDrift)")
	}
}

// Re-locking at the current content clears a previously reported drift.
//
// [scenario.engine.drift.reverify-clears]
func TestDriftReverifyClears(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "specs/models/item.md", itemSpec)
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}
	writeSpecFile(t, root, "specs/models/item.md", itemSpec+"\nedited.\n")
	if r, err := Drift(root, "web"); err != nil || !contains(r.Drifted, "domain.item") {
		t.Fatalf("precondition: spec must be drifted before re-verify, got %+v (err %v)", r, err)
	}

	// verify passing green re-invokes lock at the new content
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}
	r, err := Drift(root, "web")
	if err != nil {
		t.Fatal(err)
	}
	if contains(r.Drifted, "domain.item") || r.HasDrift() {
		t.Fatalf("re-verifying green must clear the drift, got %+v", r)
	}
	if !contains(r.Clean, "domain.item") {
		t.Errorf("a re-verified spec must be reported clean, got %+v", r)
	}
}

// A spec with no lock shard on a target is missing (unverified) — a distinct
// classification from drifted, and not the drift-gate condition.
//
// [scenario.engine.drift.never-verified-missing]
func TestDriftNeverVerifiedMissing(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "specs/models/item.md", itemSpec)

	r, err := Drift(root, "web")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(r.Missing, "domain.item") {
		t.Fatalf("a never-verified spec must be reported missing, got %+v", r)
	}
	if contains(r.Drifted, "domain.item") {
		t.Errorf("missing must be distinct from drifted, got %+v", r)
	}
	if contains(r.Clean, "domain.item") {
		t.Errorf("a never-verified spec must not be clean, got %+v", r)
	}
	if r.HasDrift() {
		t.Error("missing alone must not trip the drift gate — it is not drifted")
	}
}

// Drift consults only the content hash, never the file mtime (D7): touching
// the file must not drift it, while a one-byte content change must. The
// contrast half makes this test fail if drift ever started trusting mtimes.
//
// [scenario.engine.drift.ignores-mtime]
func TestDriftIgnoresMtime(t *testing.T) {
	root := t.TempDir()
	specPath := writeSpecFile(t, root, "specs/models/item.md", itemSpec)
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}

	for _, stamp := range []time.Time{
		time.Now().Add(1000 * time.Hour),  // fresh checkout / touch in the future
		time.Now().Add(-1000 * time.Hour), // restored backup in the past
	} {
		if err := os.Chtimes(specPath, stamp, stamp); err != nil {
			t.Fatal(err)
		}
		r, err := Drift(root, "web")
		if err != nil {
			t.Fatal(err)
		}
		if contains(r.Drifted, "domain.item") || !contains(r.Clean, "domain.item") {
			t.Fatalf("an mtime-only change (%v) must not cause drift (D7), got %+v", stamp, r)
		}
	}

	// Contrast: one changed byte at the same shape of edit IS drift. If drift
	// consulted mtimes instead of content, the loop above would have failed and
	// this would not distinguish anything; together they pin content-hash-only.
	writeSpecFile(t, root, "specs/models/item.md", itemSpec+".")
	r, err := Drift(root, "web")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(r.Drifted, "domain.item") {
		t.Fatalf("a one-byte content change must drift, got %+v", r)
	}
}
