package project

import (
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// SPEC: story.init.projection — ProjectPacks projects a stack's platform skills
// into the agent's skills dir.
func TestProjectPacks(t *testing.T) {
	root := t.TempDir()
	written, err := ProjectPacks(root, coreassets.FS, "claude", []string{"web"})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) == 0 {
		t.Fatal("no pack skills written")
	}
	mustExist(t, filepath.Join(root, ".claude", "skills", "web-development", "SKILL.md"))
	mustExist(t, filepath.Join(root, ".claude", "skills", "web-verification", "SKILL.md"))

	// unknown pack is an error, not a silent no-op
	if _, err := ProjectPacks(t.TempDir(), coreassets.FS, "claude", []string{"nope"}); err == nil {
		t.Error("expected error for unknown pack")
	}

	// codex projects to .agents/skills
	cdx := t.TempDir()
	if _, err := ProjectPacks(cdx, coreassets.FS, "codex", []string{"apple"}); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(cdx, ".agents", "skills", "ios-development", "SKILL.md"))
}
