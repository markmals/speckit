package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// [scenario.init.projection.copilot] copilot projects each command to
// .github/agents/speckit.<cmd>.agent.md and .github/prompts/speckit.<cmd>.prompt.md,
// and the orientation file is .github/copilot-instructions.md.
//
// SPEC: story.init.projection
func TestInitCopilot(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, coreassets.FS, Options{Integration: "copilot"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range initCommandSet(t) {
		mustExist(t, filepath.Join(root, ".github", "agents", "speckit."+c.Name+".agent.md"))
		mustExist(t, filepath.Join(root, ".github", "prompts", "speckit."+c.Name+".prompt.md"))
	}
	mustExist(t, filepath.Join(root, ".github", "copilot-instructions.md"))
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Error(".claude/ must not exist for copilot")
	}
}
