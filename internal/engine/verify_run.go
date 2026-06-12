package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/markmals/speckit/internal/reports"
	"github.com/markmals/speckit/internal/specmodel"
)

// VerifyConfig describes how to verify a platform: the test command to run
// (optional — empty if the report already exists), the report format and path,
// and the test source directory. Normally supplied by the platform pack's
// verify adapter.
type VerifyConfig struct {
	Command []string `json:"command,omitempty"`
	Format  string   `json:"format"` // "junit" | "swift"
	Report  string   `json:"report"` // report path, relative to root
	Source  string   `json:"source"` // test source dir, relative to root
}

// Verify runs a platform's tests (if a command is given), parses the report,
// joins outcomes to declared scenarios via source bindings, and writes the lock
// for each spec whose scenarios all passed. A spec is locked only when source
// integrity is clean (no dangling/unbound bindings, D12) and all of its
// scenarios passed (scenario.engine.lock.no-write-on-red). Only the specs the
// platform actually implements (those with a binding here) are in scope.
//
// SPEC: story.engine.verify, story.engine.lock
func Verify(root, platform string, cfg VerifyConfig) (VerifyResult, []specmodel.SpecID, error) {
	if len(cfg.Command) > 0 {
		cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)
		cmd.Dir = root
		_ = cmd.Run() // a failing suite is expected; the report carries the truth
	}

	data, err := os.ReadFile(filepath.Join(root, cfg.Report))
	if err != nil {
		return VerifyResult{}, nil, err
	}
	var results []reports.Result
	switch cfg.Format {
	case "junit":
		results, err = reports.ParseJUnit(data)
	case "swift":
		results, err = reports.ParseSwiftEvents(data)
	default:
		return VerifyResult{}, nil, fmt.Errorf("unknown report format %q", cfg.Format)
	}
	if err != nil {
		return VerifyResult{}, nil, err
	}

	bindings, err := ScanBindings(filepath.Join(root, cfg.Source))
	if err != nil {
		return VerifyResult{}, nil, err
	}

	specs, err := specmodel.LoadLibrary(os.DirFS(root))
	if err != nil {
		return VerifyResult{}, nil, err
	}
	scenarioSpec := map[specmodel.SpecID]specmodel.SpecID{}
	specScenarios := map[specmodel.SpecID][]specmodel.SpecID{}
	for _, s := range specs {
		for _, sc := range s.Scenarios {
			if sc.SubID == "" {
				continue
			}
			id := specmodel.SpecID(sc.SubID)
			scenarioSpec[id] = s.ID
			specScenarios[s.ID] = append(specScenarios[s.ID], id)
		}
	}

	// A spec is in scope for this platform if any of its scenarios is bound here.
	inScope := map[specmodel.SpecID]bool{}
	for _, b := range bindings {
		if spec, ok := scenarioSpec[b.Scenario]; ok {
			inScope[spec] = true
		}
	}
	declared := map[specmodel.SpecID]bool{}
	for spec := range inScope {
		for _, sc := range specScenarios[spec] {
			declared[sc] = true
		}
	}

	v := Join(declared, results, bindings)

	var locked []specmodel.SpecID
	if len(v.Dangling) == 0 && len(v.Unbound) == 0 { // source integrity clean
		passed := map[specmodel.SpecID]bool{}
		for _, sc := range v.Passed {
			passed[sc] = true
		}
		for spec := range inScope {
			all := len(specScenarios[spec]) > 0
			for _, sc := range specScenarios[spec] {
				if !passed[sc] {
					all = false
					break
				}
			}
			if all {
				if err := Lock(root, platform, spec); err != nil {
					return v, locked, err
				}
				locked = append(locked, spec)
			}
		}
	}
	sortIDs(locked)
	return v, locked, nil
}
