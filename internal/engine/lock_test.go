package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashDeterministicAndContentSensitive(t *testing.T) {
	a := Hash([]byte("hello"))
	if a != Hash([]byte("hello")) {
		t.Error("hash must be deterministic")
	}
	if a == Hash([]byte("hello!")) {
		t.Error("hash must change with content")
	}
}

// SPEC: domain.lock (L1 sole writer; L3 sharded)
func TestWriteReadShard(t *testing.T) {
	root := t.TempDir()
	want := Shard{SpecHash: "abc", Scenarios: map[string]string{"scenario.x.y": "pass"}}
	if err := WriteShard(root, "web", "domain.item", want); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ReadShard(root, "web", "domain.item")
	if err != nil || !ok {
		t.Fatalf("read: ok=%v err=%v", ok, err)
	}
	if got.SpecHash != "abc" || got.Scenarios["scenario.x.y"] != "pass" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".speckit", "lock", "web", "domain.item.json")); err != nil {
		t.Errorf("shard not at the expected sharded path: %v", err)
	}
	if _, ok, _ := ReadShard(root, "web", "nope"); ok {
		t.Error("expected exists=false for an absent shard")
	}
}
