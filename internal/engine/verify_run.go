package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/markmals/speckit/internal/reports"
	"github.com/markmals/speckit/internal/specmodel"
)

// VerifyConfig describes how to verify a platform: the test command to run as a
// shell string (optional — empty if the report already exists, à la a Mise
// task's `run`), the report format and path, and the test source directory.
// Normally supplied by the platform pack's verify adapter.
type VerifyConfig struct {
	Command string `json:"command,omitempty"`
	Format  string `json:"format"` // "junit" | "swift"
	Report  string `json:"report"` // report path, relative to root
	Source  string `json:"source"` // test source dir, relative to root
}

// joinPlatform runs the platform's command (if any), parses the report, scans
// source bindings, scopes to the specs the platform implements (those with a
// binding here), and joins. Shared by Verify (which then locks the green specs)
// and Parity (which crosses the result with deviation markers).
func joinPlatform(root string, cfg VerifyConfig) (VerifyResult, map[specmodel.SpecID]bool, map[specmodel.SpecID][]specmodel.SpecID, error) {
	if cfg.Command != "" {
		// cfg.Command is a shell string from the project's own .speckit/verify
		// config (developer-controlled, like a Mise task's `run`), so shell
		// interpretation is intended — the project owner is the trust boundary.
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", cfg.Command)
		} else {
			cmd = exec.Command("sh", "-c", cfg.Command)
		}
		cmd.Dir = root
		_ = cmd.Run() // a failing suite is expected; the report carries the truth
	}

	data, err := os.ReadFile(filepath.Join(root, cfg.Report))
	if err != nil {
		return VerifyResult{}, nil, nil, err
	}
	var results []reports.Result
	switch cfg.Format {
	case "junit":
		results, err = reports.ParseJUnit(data)
	case "swift":
		results, err = reports.ParseSwiftEvents(data)
	default:
		return VerifyResult{}, nil, nil, fmt.Errorf("unknown report format %q", cfg.Format)
	}
	if err != nil {
		return VerifyResult{}, nil, nil, err
	}

	bindings, err := ScanBindings(filepath.Join(root, cfg.Source))
	if err != nil {
		return VerifyResult{}, nil, nil, err
	}

	specs, err := specmodel.LoadLibrary(os.DirFS(root))
	if err != nil {
		return VerifyResult{}, nil, nil, err
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

	return Join(declared, results, bindings), inScope, specScenarios, nil
}

// Verify runs a platform's tests, joins outcomes to declared scenarios, and
// writes the lock for each spec whose scenarios all passed — only when source
// integrity is clean (no dangling/unbound, D12) and all of the spec's scenarios
// passed (scenario.engine.lock.no-write-on-red).
//
// SPEC: story.engine.verify, story.engine.lock
func Verify(root, platform string, cfg VerifyConfig) (VerifyResult, []specmodel.SpecID, error) {
	v, inScope, specScenarios, err := joinPlatform(root, cfg)
	if err != nil {
		return VerifyResult{}, nil, err
	}

	var locked []specmodel.SpecID
	if len(v.Dangling) == 0 && len(v.Unbound) == 0 {
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
