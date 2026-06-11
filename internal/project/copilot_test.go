package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// SPEC: story.init.projection (scenario.init.projection.copilot)
func TestInitCopilot(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, coreassets.FS, Options{Integration: "copilot"}); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, ".github", "agents", "speckit.specify.agent.md"))
	mustExist(t, filepath.Join(root, ".github", "prompts", "speckit.specify.prompt.md"))
	mustExist(t, filepath.Join(root, ".github", "copilot-instructions.md"))
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Error(".claude/ must not exist for copilot")
	}
}
