package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// SPEC: story.init.basic
// SPEC: story.init.projection
func TestInitClaude(t *testing.T) {
	root := t.TempDir()
	written, err := Init(root, coreassets.FS, Options{Integration: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) == 0 {
		t.Fatal("no paths written")
	}

	// [scenario.init.projection.claude] claude projects to .claude/skills + CLAUDE.md
	t.Run("claude projection", func(t *testing.T) {
		mustExist(t, filepath.Join(root, ".claude", "skills", "speckit-specify", "SKILL.md"))
		mustExist(t, filepath.Join(root, ".claude", "skills", "speckit-plan", "SKILL.md"))
		mustExist(t, filepath.Join(root, "CLAUDE.md"))
	})

	// [scenario.init.projection.fork-divergence] .speckit not .specify; no scripts; no .specify refs
	t.Run("fork divergence", func(t *testing.T) {
		mustExist(t, filepath.Join(root, ".speckit"))
		if _, err := os.Stat(filepath.Join(root, ".specify")); !os.IsNotExist(err) {
			t.Error(".specify/ must not exist (D6)")
		}
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, _ error) error {
			if strings.HasSuffix(p, ".sh") || strings.HasSuffix(p, ".ps1") {
				t.Errorf("unexpected runtime script (D2): %s", p)
			}
			return nil
		})
		b, _ := os.ReadFile(filepath.Join(root, ".claude", "skills", "speckit-specify", "SKILL.md"))
		if strings.Contains(string(b), ".specify/") {
			t.Error("projected command still references .specify/ (D6)")
		}
	})
}

// [scenario.init.basic.non-empty-guard] refuse a non-empty dir without --force
//
// SPEC: story.init.basic
func TestInitNonEmptyGuard(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(dir, coreassets.FS, Options{Integration: "claude"}); err == nil {
		t.Error("expected refusal on non-empty dir without --force")
	}
	if _, err := Init(dir, coreassets.FS, Options{Integration: "claude", Force: true}); err != nil {
		t.Errorf("--force should proceed on non-empty dir: %v", err)
	}
}

// SPEC: story.init.basic
func TestInitUnknownIntegration(t *testing.T) {
	if _, err := Init(t.TempDir(), coreassets.FS, Options{Integration: "nope"}); err == nil {
		t.Error("expected error for unknown integration")
	}
}

// SPEC: story.init.projection — init projects the process-discipline skills
// into the agent's skills dir (the process-pack).
func TestInitProjectsSkills(t *testing.T) {
	claude := t.TempDir()
	if _, err := Init(claude, coreassets.FS, Options{Integration: "claude"}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"test-driven-development", "verification-before-completion", "adversarial-review", "systematic-debugging"} {
		mustExist(t, filepath.Join(claude, ".claude", "skills", s, "SKILL.md"))
	}

	codex := t.TempDir()
	if _, err := Init(codex, coreassets.FS, Options{Integration: "codex"}); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(codex, ".agents", "skills", "test-driven-development", "SKILL.md"))

	// copilot has no skills dir — it must not get a skills/ tree.
	cop := t.TempDir()
	if _, err := Init(cop, coreassets.FS, Options{Integration: "copilot"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cop, ".claude", "skills")); !os.IsNotExist(err) {
		t.Error("copilot must not get a .claude/skills tree")
	}
}

func mustExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Errorf("missing %s: %v", p, err)
	}
}
