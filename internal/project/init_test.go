package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
)

// TestInitRecordsAgent — init seeds .speckit/specs.json with the chosen agent
// (so projected guidance knows which agent owns the runtime dirs) without the
// user hand-editing specs.json. AddTarget must then preserve it.
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
			if err := config.AddTarget(root, "app", config.Target{Dir: "app", Format: "swift", Report: "r.ndjson", Source: config.SourcePaths{"app"}}); err != nil {
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

// [scenario.init.projection.claude] claude projects the command set to
// .claude/skills/speckit-<cmd>/SKILL.md, the orientation file is CLAUDE.md, and
// the install is recorded: Init returns the manifest of every installed file
// and .speckit/specs.json records the integration.
//
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

	for _, c := range initCommandSet(t) {
		mustExist(t, filepath.Join(root, ".claude", "skills", "speckit-"+c.Name, "SKILL.md"))
	}
	mustExist(t, filepath.Join(root, "CLAUDE.md"))

	// The integration manifest records what was installed: the returned paths
	// and the on-disk tree are the same set, and specs.json records the agent.
	returned := map[string]bool{}
	for _, p := range written {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatal(err)
		}
		returned[filepath.ToSlash(rel)] = true
	}
	onDisk := strings.Split(strings.TrimSpace(manifestOf(t, root)), "\n")
	if len(returned) != len(onDisk) {
		t.Errorf("returned manifest has %d entries, %d files on disk", len(returned), len(onDisk))
	}
	for _, f := range onDisk {
		if !returned[f] {
			t.Errorf("installed file missing from returned manifest: %s", f)
		}
	}
	cfg, found, err := config.Load(root)
	if err != nil || !found {
		t.Fatalf("specs.json not recorded: found=%v err=%v", found, err)
	}
	if cfg.Agent != "claude" {
		t.Errorf("recorded integration = %q, want claude", cfg.Agent)
	}
}

// [scenario.init.projection.fork-divergence] the runtime dir is .speckit/,
// never .specify/ (D6); no runtime scripts are written — command logic lives in
// the specify binary, and init writes no git-hook trampolines today, so zero
// scripts full stop (D2); no workflow engine / workflows/ dir is installed; and
// no "GitHub Spec Kit" banner appears in anything init produces (D1). Every
// projected file's *content* is scanned, not just the tree shape.
//
// SPEC: story.init.projection
func TestInitForkDivergence(t *testing.T) {
	for _, integ := range []string{"claude", "codex", "copilot", "generic"} {
		t.Run(integ, func(t *testing.T) {
			root := t.TempDir()
			if _, err := Init(root, coreassets.FS, Options{Integration: integ}); err != nil {
				t.Fatal(err)
			}
			mustExist(t, filepath.Join(root, ".speckit"))
			if _, err := os.Stat(filepath.Join(root, ".specify")); !os.IsNotExist(err) {
				t.Error(".specify/ must not exist (D6)")
			}
			err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if d.Name() == "workflows" {
						t.Errorf("workflow engine dir installed (deferred): %s", p)
					}
					return nil
				}
				switch strings.ToLower(filepath.Ext(p)) {
				case ".sh", ".bash", ".zsh", ".ps1", ".cmd", ".bat":
					t.Errorf("unexpected runtime script (D2): %s", p)
				}
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				if strings.Contains(string(b), ".specify/") {
					t.Errorf("projected content references .specify/ (D6): %s", p)
				}
				if strings.Contains(string(b), "GitHub Spec Kit") {
					t.Errorf("upstream banner in projected content (D1): %s", p)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// [scenario.init.basic.fresh] a fresh init in an empty directory writes the
// chosen agent's command prompts in that agent's native format, the spec
// conventions (the .speckit/ runtime: constitution + templates) and the process
// prompts (the process-pack skills), and every generated command prompt invokes
// the specify binary — with zero runtime scripts on disk (D2; init writes no
// git-hook trampolines today).
//
// SPEC: story.init.basic
func TestInitFresh(t *testing.T) {
	invokesSpecify := regexp.MustCompile(`specify (scan|verify|drift|cover|parity|gate|work)`)
	commands := initCommandSet(t)
	prompts := map[string]func(name string) []string{
		"claude": func(n string) []string {
			return []string{filepath.Join(".claude", "skills", "speckit-"+n, "SKILL.md")}
		},
		"codex": func(n string) []string {
			return []string{filepath.Join(".agents", "skills", "speckit-"+n, "SKILL.md")}
		},
		"copilot": func(n string) []string {
			return []string{
				filepath.Join(".github", "agents", "speckit."+n+".agent.md"),
				filepath.Join(".github", "prompts", "speckit."+n+".prompt.md"),
			}
		},
	}
	for _, agent := range []string{"claude", "codex", "copilot"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir() // an empty directory: no --force needed
			if _, err := Init(root, coreassets.FS, Options{Integration: agent}); err != nil {
				t.Fatal(err)
			}

			// Command prompts in the agent's native format, each invoking specify.
			for _, c := range commands {
				for _, rel := range prompts[agent](c.Name) {
					p := filepath.Join(root, rel)
					mustExist(t, p)
					b, err := os.ReadFile(p)
					if err != nil {
						t.Fatal(err)
					}
					if !invokesSpecify.Match(b) {
						t.Errorf("command prompt does not invoke specify: %s", rel)
					}
				}
			}

			// Spec conventions: the .speckit/ runtime.
			for _, rel := range []string{
				".speckit/memory/constitution.md",
				".speckit/templates/spec-template.md",
				".speckit/templates/plan-template.md",
				".speckit/templates/tasks-template.md",
				".speckit/templates/checklist-template.md",
			} {
				mustExist(t, filepath.Join(root, filepath.FromSlash(rel)))
			}

			// Process prompts: the process-pack skills in the agent's skills dir.
			skillsDir := AdapterMust(t, agent).SkillsDir()
			for _, s := range []string{"test-driven-development", "verification-before-completion", "adversarial-review", "systematic-debugging"} {
				mustExist(t, filepath.Join(root, filepath.FromSlash(skillsDir), s, "SKILL.md"))
			}

			// Zero runtime scripts.
			err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				switch strings.ToLower(filepath.Ext(p)) {
				case ".sh", ".bash", ".zsh", ".ps1", ".cmd", ".bat":
					t.Errorf("runtime script written by init (D2): %s", p)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// initCommandSet loads the authored command set from the embedded assets — the
// same input Init projects — so tests can quantify over every command.
func initCommandSet(t *testing.T) []Command {
	t.Helper()
	cmds, err := loadCommands(coreassets.FS)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) == 0 {
		t.Fatal("no authored commands in coreassets")
	}
	return cmds
}

// [scenario.init.basic.non-empty-guard] refuse a non-empty dir without --force
// (writing nothing); with --force proceed, leaving unrelated files untouched.
//
// SPEC: story.init.basic
func TestInitNonEmptyGuard(t *testing.T) {
	dir := t.TempDir()
	unrelated := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(unrelated, []byte("precious user bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(dir, coreassets.FS, Options{Integration: "claude"}); err == nil {
		t.Error("expected refusal on non-empty dir without --force")
	}
	// The refusal must not have started writing.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "existing.txt" {
		t.Errorf("refused init still wrote into the directory: %v", entries)
	}
	if _, err := Init(dir, coreassets.FS, Options{Integration: "claude", Force: true}); err != nil {
		t.Errorf("--force should proceed on non-empty dir: %v", err)
	}
	got, err := os.ReadFile(unrelated)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "precious user bytes" {
		t.Errorf("--force init touched an unrelated file: %q", got)
	}
	mustExist(t, filepath.Join(dir, "CLAUDE.md"))
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
