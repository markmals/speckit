package engine

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/markmals/speckit/internal/specmodel"
)

var deviateRe = regexp.MustCompile(`// SPEC: (scenario\.[a-z0-9.\-]+) \(deviates: ([^)]*)\)`)

// ScanDeviations reads scenario-scoped deviation markers from a target's
// source — `// SPEC: <scenario-id> (deviates: <reason>)` (CONVENTIONS) — and
// returns scenario-id -> reason.
//
// SPEC: story.engine.parity
func ScanDeviations(dir string) (map[specmodel.SpecID]string, error) {
	out := map[specmodel.SpecID]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range deviateRe.FindAllStringSubmatch(string(b), -1) {
			out[specmodel.SpecID(m[1])] = m[2]
		}
		return nil
	})
	if os.IsNotExist(err) {
		return out, nil
	}
	return out, err
}

// ParityCell is a scenario's state on a target (D11).
type ParityCell struct {
	Scenario specmodel.SpecID `json:"scenario"`
	State    string           `json:"state"` // conforming | declared-deviation | drifted | suspect | missing
	Reason   string           `json:"reason,omitempty"`
}

// ParityReport is a target's parity matrix.
type ParityReport struct {
	Target string       `json:"target"`
	Cells    []ParityCell `json:"cells"`
}

// Gated reports whether `parity --gate` should fail: any cell that is not
// conforming. A declared-deviation is human-attested and never auto-green (D11);
// drifted/suspect/missing are problems — so only an all-conforming matrix passes.
//
// SPEC: story.engine.parity (scenario.engine.parity.suspect-lying-marker)
func (r ParityReport) Gated() bool {
	for _, c := range r.Cells {
		if c.State != "conforming" {
			return true
		}
	}
	return false
}

// Parity crosses the verify outcome with deviation markers into the five-state
// matrix (D11). Deviation-presence and test-outcome are computed on independent
// axes, so a marker over a failing test is suspect, never declared-deviation.
//
// SPEC: story.engine.parity
func Parity(root, target string, cfg VerifyConfig) (ParityReport, error) {
	v, _, _, err := joinTarget(root, cfg)
	if err != nil {
		return ParityReport{}, err
	}
	deviations, err := ScanDeviations(filepath.Join(root, cfg.Source))
	if err != nil {
		return ParityReport{}, err
	}

	report := ParityReport{Target: target}
	cell := func(s specmodel.SpecID, passed, failed bool) ParityCell {
		reason, hasDev := deviations[s]
		switch {
		case passed && hasDev:
			return ParityCell{Scenario: s, State: "declared-deviation", Reason: reason}
		case passed:
			return ParityCell{Scenario: s, State: "conforming"}
		case failed && hasDev:
			return ParityCell{Scenario: s, State: "suspect", Reason: reason}
		case failed:
			return ParityCell{Scenario: s, State: "drifted"}
		default:
			return ParityCell{Scenario: s, State: "missing"}
		}
	}
	for _, s := range v.Passed {
		report.Cells = append(report.Cells, cell(s, true, false))
	}
	for _, s := range v.Failed {
		report.Cells = append(report.Cells, cell(s, false, true))
	}
	for _, s := range v.Unjoinable {
		report.Cells = append(report.Cells, cell(s, false, false))
	}
	sort.Slice(report.Cells, func(i, j int) bool { return report.Cells[i].Scenario < report.Cells[j].Scenario })
	return report, nil
}
