package engine

import (
	"io/fs"
	"os"

	"github.com/markmals/speckit/internal/specmodel"
)

// DriftReport classifies every spec in the library on a target.
//
// SPEC: story.engine.drift
type DriftReport struct {
	Clean   []specmodel.SpecID `json:"clean"`
	Drifted []specmodel.SpecID `json:"drifted"`
	Missing []specmodel.SpecID `json:"missing"`
}

// HasDrift reports whether any spec drifted — the gate / non-zero-exit condition.
func (r DriftReport) HasDrift() bool { return len(r.Drifted) > 0 }

// Drift compares each spec's current content hash to its locked-green hash on a
// target: a hash mismatch is drifted, an absent shard is missing, a match is
// clean. It never consults mtimes (D7).
//
// SPEC: story.engine.drift
func Drift(root, target string) (DriftReport, error) {
	fsys := os.DirFS(root)
	specs, err := specmodel.LoadLibrary(fsys)
	if err != nil {
		return DriftReport{}, err
	}
	report := DriftReport{
		Clean:   []specmodel.SpecID{},
		Drifted: []specmodel.SpecID{},
		Missing: []specmodel.SpecID{},
	}
	for _, s := range specs {
		content, err := fs.ReadFile(fsys, s.Path)
		if err != nil {
			return DriftReport{}, err
		}
		shard, ok, err := ReadShard(root, target, s.ID)
		if err != nil {
			return DriftReport{}, err
		}
		switch {
		case !ok:
			report.Missing = append(report.Missing, s.ID)
		case shard.SpecHash != Hash(content):
			report.Drifted = append(report.Drifted, s.ID)
		default:
			report.Clean = append(report.Clean, s.ID)
		}
	}
	return report, nil
}
