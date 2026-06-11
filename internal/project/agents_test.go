package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// SPEC: story.init.projection (scenario.init.projection.codex, scenario.init.projection.generic)
func TestInitAgentsAdapters(t *testing.T) {
	for _, agent := range []string{"codex", "generic"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			if _, err := Init(root, coreassets.FS, Options{Integration: agent}); err != nil {
				t.Fatal(err)
			}
			mustExist(t, filepath.Join(root, ".agents", "skills", "speckit-specify", "SKILL.md"))
			mustExist(t, filepath.Join(root, "AGENTS.md"))
			// codex and generic both use .agents/ + AGENTS.md — never .claude/.
			if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
				t.Errorf(".claude/ must not exist for %s", agent)
			}
		})
	}
}
