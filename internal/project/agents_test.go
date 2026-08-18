package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// [scenario.init.projection.codex] codex projects the command set to
// .agents/skills/speckit-<cmd>/SKILL.md and the orientation file is AGENTS.md —
// the shared Codex/Copilot substrate (D4). generic rides the same adapter
// family and is asserted alongside; the codex==generic tree equivalence itself
// is proven by TestInitGenericMatchesCodex.
//
// SPEC: story.init.projection
func TestInitAgentsAdapters(t *testing.T) {
	for _, agent := range []string{"codex", "generic"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			if _, err := Init(root, coreassets.FS, Options{Integration: agent}); err != nil {
				t.Fatal(err)
			}
			for _, c := range initCommandSet(t) {
				mustExist(t, filepath.Join(root, ".agents", "skills", "speckit-"+c.Name, "SKILL.md"))
			}
			mustExist(t, filepath.Join(root, "AGENTS.md"))
			// codex and generic both use .agents/ + AGENTS.md — never .claude/.
			if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
				t.Errorf(".claude/ must not exist for %s", agent)
			}
		})
	}
}
