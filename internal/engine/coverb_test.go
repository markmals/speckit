package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A story spec with one scenario id, so lock shards carry covered scenarios.
const coverBSpec = `---
id: story.covb.x
kind: story
---

# Story

# Acceptance Criteria

## Scenario 1: a

<!-- id: scenario.covb.x.a -->

- Given a
`

// Cover derives green from the lock shard alone: even when the target's
// configured test command is a booby trap (it would drop a sentinel file and
// fail), Cover reports conforming and the sentinel never appears — so any
// regression that made Cover re-run tests turns this red.
//
// [scenario.engine.cover.reads-lock]
func TestCoverReadsLock(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "specs/models/item.md", itemSpec)
	if err := Lock(root, "web", "domain.item"); err != nil {
		t.Fatal(err)
	}
	// A real config wiring web to a command that records having run and fails.
	sentinel := "cover-ran.txt"
	cfg := `{
  "version": 1,
  "paths": {"specs": "specs", "features": "features"},
  "targets": {
    "web": {
      "dir": ".",
      "command": "sh -c 'echo ran > ` + sentinel + `; exit 1'",
      "format": "junit",
      "report": "report.junit.xml",
      "source": ["src"]
    }
  }
}
`
	writeSpecFile(t, root, ".speckit/specs.json", cfg)

	r, err := Cover(root, "domain.item")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Cells) != 1 || r.Cells[0].Target != "web" || r.Cells[0].State != "conforming" {
		t.Fatalf("green must be derived from the lock, got %+v", r.Cells)
	}
	if _, err := os.Stat(filepath.Join(root, sentinel)); !os.IsNotExist(err) {
		t.Fatal("cover ran the target's test command — green must come from the lock, not a re-run")
	}
	if _, err := os.Stat(filepath.Join(root, "report.junit.xml")); !os.IsNotExist(err) {
		t.Fatal("cover produced a test report — it must not re-run tests")
	}
}

// The JSON projection of a cover report is a structured per-target record:
// the spec, and one cell per target carrying target, state, and the covered
// scenarios read from that target's lock shard.
//
// [scenario.engine.cover.json]
func TestCoverJSONShape(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-covb/stories/covb.x.md", coverBSpec)
	if err := Lock(root, "web", "story.covb.x"); err != nil {
		t.Fatal(err)
	}

	r, err := Cover(root, "story.covb.x")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Spec  string `json:"spec"`
		Cells []struct {
			Target    string            `json:"target"`
			State     string            `json:"state"`
			Scenarios map[string]string `json:"scenarios"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Spec != "story.covb.x" {
		t.Errorf("spec: got %q, want %q", got.Spec, "story.covb.x")
	}
	if len(got.Cells) != 1 {
		t.Fatalf("expected one per-target record, got %s", b)
	}
	c := got.Cells[0]
	if c.Target != "web" || c.State != "conforming" {
		t.Errorf("cell must carry target and state, got %+v", c)
	}
	if c.Scenarios["scenario.covb.x.a"] != "pass" {
		t.Errorf("cell must carry the covered scenarios, got %+v", c.Scenarios)
	}
}
