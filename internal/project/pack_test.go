package project

import (
	"os"
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

	// a real scaffold that ships no pack (library stacks) is NOT an error — it just
	// projects no skills, so `specify packs` doesn't fail on a packless target.
	noPack, err := ProjectPacks(t.TempDir(), coreassets.FS, "claude", []string{"swift-package"})
	if err != nil {
		t.Errorf("packless stack must not error: %v", err)
	}
	if len(noPack) != 0 {
		t.Errorf("packless stack must project no skills, got %d", len(noPack))
	}

	// codex projects to .agents/skills; the apple pack carries the cross-platform
	// ios-* skills plus the AppKit (macOS) skill suite adapted from mac-dev-skills.
	cdx := t.TempDir()
	appleWritten, err := ProjectPacks(cdx, coreassets.FS, "codex", []string{"apple"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"ios-development", "ios-simulator-control",
		"appkit-design", "appkit-setup", "appkit-ui-testing",
		"appkit-packaging", "appkit-migration", "appkit-code-review",
	} {
		mustExist(t, filepath.Join(cdx, ".agents", "skills", name, "SKILL.md"))
	}
	if len(appleWritten) < 16 {
		t.Errorf("apple pack projected %d skills, want the full AppKit suite (>=16)", len(appleWritten))
	}

	// claude gets the apple pack with full directory DEPTH (references/ survive the
	// projection) plus the stack agent — claude has an AgentsDir; codex/generic/copilot
	// skip stack agents (their AgentsDir is "").
	cl := t.TempDir()
	if _, err := ProjectPacks(cl, coreassets.FS, "claude", []string{"apple"}); err != nil {
		t.Fatal(err)
	}
	// whole-directory projection: a deepened skill's references/ land beside its SKILL.md.
	mustExist(t, filepath.Join(cl, ".claude", "skills", "appkit-design", "SKILL.md"))
	mustExist(t, filepath.Join(cl, ".claude", "skills", "appkit-design", "references", "semantic-color.md"))
	// the offline HIG corpus (apple-hig) projects its whole tree, including the index.
	mustExist(t, filepath.Join(cl, ".claude", "skills", "apple-hig", "references", "hig", "INDEX.md"))
	// the per-stack agent lands in claude's agents dir.
	mustExist(t, filepath.Join(cl, ".claude", "agents", "appkit-dev.md"))

	// codex has no agents dir, so the stack agent is skipped there (skills still land).
	if _, err := os.Stat(filepath.Join(cdx, ".agents", "agents", "appkit-dev.md")); !os.IsNotExist(err) {
		t.Error("codex (no AgentsDir) must not receive the stack agent")
	}
}
