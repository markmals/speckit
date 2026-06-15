package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
)

// TestInitRecordsAgent — init seeds .speckit/specs.json with the chosen agent, so
// `target add` / `packs` (gated on a recorded agent) can project the stack packs
// without the user hand-editing specs.json. AddTarget must then preserve it.
func TestInitRecordsAgent(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "copilot", "generic"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			if _, err := Init(root, coreassets.FS, Options{Integration: agent}); err != nil {
				t.Fatal(err)
			}
			cfg, found, err := config.Load(root)
			if err != nil || !found {
				t.Fatalf("specs.json not written by init: found=%v err=%v", found, err)
			}
			if cfg.Agent != agent {
				t.Errorf("recorded agent = %q, want %q", cfg.Agent, agent)
			}
			if cfg.Paths.Specs != "specs" || cfg.Paths.Features != "features" {
				t.Errorf("default paths not seeded: %+v", cfg.Paths)
			}

			// target add must preserve the agent (it load-modify-saves the config).
			if err := config.AddTarget(root, "app", config.Target{Stack: "apple", Format: "swift"}); err != nil {
				t.Fatal(err)
			}
			after, _, _ := config.Load(root)
			if after.Agent != agent {
				t.Errorf("agent lost after target add: %q, want %q", after.Agent, agent)
			}
			if _, ok := after.Targets["app"]; !ok {
				t.Error("target not added")
			}
		})
	}
}

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

// SPEC: story.init.projection — claude gets the review subagents (the
// claude-pack); codex/generic/copilot have no projectable subagent dir.
func TestInitProjectsSubagents(t *testing.T) {
	claude := t.TempDir()
	if _, err := Init(claude, coreassets.FS, Options{Integration: "claude"}); err != nil {
		t.Fatal(err)
	}
	for _, a := range []string{"spec-reviewer", "test-gap-finder", "drift-hunter", "handoff-builder"} {
		mustExist(t, filepath.Join(claude, ".claude", "agents", a+".md"))
	}

	for _, integ := range []string{"codex", "generic", "copilot"} {
		dir := t.TempDir()
		if _, err := Init(dir, coreassets.FS, Options{Integration: integ}); err != nil {
			t.Fatal(err)
		}
		_ = filepath.WalkDir(dir, func(p string, _ os.DirEntry, _ error) error {
			if strings.Contains(p, "spec-reviewer.md") {
				t.Errorf("%s must not get review subagents: %s", integ, p)
			}
			return nil
		})
	}
}

func mustExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Errorf("missing %s: %v", p, err)
	}
}
