package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/markmals/speckit/internal/specmodel"
)

// CoverCell is a spec's state on one platform, derived from the lock.
type CoverCell struct {
	Platform  string            `json:"platform"`
	State     string            `json:"state"` // conforming | drifted | missing
	Scenarios map[string]string `json:"scenarios,omitempty"`
}

// CoverReport is one spec's coverage across the platforms that have lock state.
//
// SPEC: story.engine.cover
type CoverReport struct {
	Spec  specmodel.SpecID `json:"spec"`
	Cells []CoverCell      `json:"cells"`
}

// Cover reports a spec's state on each platform, read from the lock (no test
// re-run): a matching hash is conforming, a stale hash is drifted, an absent
// shard is missing.
//
// SPEC: story.engine.cover
func Cover(root string, id specmodel.SpecID) (CoverReport, error) {
	fsys := os.DirFS(root)
	specs, err := specmodel.LoadLibrary(fsys)
	if err != nil {
		return CoverReport{}, err
	}
	var found *specmodel.Spec
	for i := range specs {
		if specs[i].ID == id {
			found = &specs[i]
			break
		}
	}
	if found == nil {
		return CoverReport{}, fmt.Errorf("spec %q not found in the library", id)
	}
	content, err := fs.ReadFile(fsys, found.Path)
	if err != nil {
		return CoverReport{}, err
	}
	current := Hash(content)

	platforms, err := lockPlatforms(root)
	if err != nil {
		return CoverReport{}, err
	}
	report := CoverReport{Spec: id, Cells: []CoverCell{}}
	for _, p := range platforms {
		shard, ok, err := ReadShard(root, p, id)
		if err != nil {
			return CoverReport{}, err
		}
		cell := CoverCell{Platform: p}
		switch {
		case !ok:
			cell.State = "missing"
		case shard.SpecHash != current:
			cell.State, cell.Scenarios = "drifted", shard.Scenarios
		default:
			cell.State, cell.Scenarios = "conforming", shard.Scenarios
		}
		report.Cells = append(report.Cells, cell)
	}
	return report, nil
}

// lockPlatforms lists the platforms that have lock state (subdirs of
// .speckit/lock/).
func lockPlatforms(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, ".speckit", "lock"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}
