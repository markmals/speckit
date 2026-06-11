package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/markmals/speckit/internal/specmodel"
)

// Shard is one acknowledgment-lock file at
// .speckit/lock/<platform>/<spec-id>.json — the content hash of the spec
// version last verified/acked green on a platform, plus per-scenario results.
//
// SPEC: domain.lock
type Shard struct {
	SpecHash   string            `json:"spec_hash"`
	Scenarios  map[string]string `json:"scenarios"` // scenario-id -> "pass" | "fail"
	VerifiedAt string            `json:"verified_at,omitempty"`
}

// Hash returns the content hash of a spec's bytes — L2: content, never mtime.
func Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func shardPath(root, platform string, id specmodel.SpecID) string {
	return filepath.Join(root, ".speckit", "lock", platform, string(id)+".json")
}

// WriteShard writes the shard for (platform, id). `specify lock` is the sole
// writer of drift state (L1); shards are one-file-per-spec (L3).
func WriteShard(root, platform string, id specmodel.SpecID, shard Shard) error {
	p := shardPath(root, platform, id)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(shard, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o644)
}

// ReadShard reads the shard for (platform, id); exists is false if absent.
func ReadShard(root, platform string, id specmodel.SpecID) (shard Shard, exists bool, err error) {
	b, err := os.ReadFile(shardPath(root, platform, id))
	if os.IsNotExist(err) {
		return Shard{}, false, nil
	}
	if err != nil {
		return Shard{}, false, err
	}
	if err := json.Unmarshal(b, &shard); err != nil {
		return Shard{}, false, err
	}
	return shard, true, nil
}

// Lock acknowledges a spec as green on a platform at its current content,
// writing the lock shard. Normally invoked by `specify verify` on green with
// real per-scenario results; invoked directly it records the spec's scenarios
// as acknowledged.
//
// SPEC: story.engine.lock
func Lock(root, platform string, id specmodel.SpecID) error {
	fsys := os.DirFS(root)
	specs, err := specmodel.LoadLibrary(fsys)
	if err != nil {
		return err
	}
	var found *specmodel.Spec
	for i := range specs {
		if specs[i].ID == id {
			found = &specs[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("spec %q not found in the library", id)
	}
	content, err := fs.ReadFile(fsys, found.Path)
	if err != nil {
		return err
	}
	scenarios := map[string]string{}
	for _, sc := range found.Scenarios {
		if sc.SubID != "" {
			scenarios[sc.SubID] = "pass"
		}
	}
	return WriteShard(root, platform, id, Shard{SpecHash: Hash(content), Scenarios: scenarios})
}
