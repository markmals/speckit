package engine

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The offline guarantee is structural: the engine packages must not — even
// transitively — import a work provider or the GitHub client, so every engine
// command runs with no network and no credentials. `go list -deps` computes
// the transitive import closure; any forbidden package appearing there fails.
//
// SPEC: story.work.providers (scenario.work.providers.import-firewall)
// [scenario.work.providers.import-firewall]
func TestEngineImportFirewall(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go binary unavailable, cannot compute the import closure: %v", err)
	}

	// repo root, located relative to this source file (not the cwd)
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this source file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	enginePkgs := []string{
		"./internal/engine",
		"./internal/specmodel",
		"./internal/reports",
		"./internal/config",
	}
	forbidden := []string{
		"github.com/markmals/speckit/internal/github",
		"github.com/markmals/speckit/internal/work",
	}

	cmd := exec.Command(goBin, append([]string{"list", "-deps"}, enginePkgs...)...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, out)
	}

	deps := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(deps) == 0 || deps[0] == "" {
		t.Fatal("go list -deps returned no packages")
	}
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		for _, bad := range forbidden {
			if dep == bad || strings.HasPrefix(dep, bad+"/") {
				t.Errorf("engine import firewall breached: %s is in the transitive closure of %v", dep, enginePkgs)
			}
		}
	}
}
