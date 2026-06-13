package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// SPEC: story.init.projection — init seeds each agent's repo-local memory/ store
// and wires the orientation file to load its index.
func TestInitProjectsMemory(t *testing.T) {
	cases := []struct {
		integration string
		memory      string // expected seed path
		orient      string // orientation file that must reference the index
		directive   string // text the orientation file must contain
	}{
		{"claude", ".claude/memory/MEMORY.md", "CLAUDE.md", "@.claude/memory/MEMORY.md"},
		{"codex", ".agents/memory/MEMORY.md", "AGENTS.md", ".agents/memory/MEMORY.md"},
		{"generic", ".agents/memory/MEMORY.md", "AGENTS.md", ".agents/memory/MEMORY.md"},
		{"copilot", ".github/memory/MEMORY.md", ".github/copilot-instructions.md", ".github/memory/MEMORY.md"},
	}
	for _, tc := range cases {
		t.Run(tc.integration, func(t *testing.T) {
			root := t.TempDir()
			if _, err := Init(root, coreassets.FS, Options{Integration: tc.integration}); err != nil {
				t.Fatal(err)
			}
			mustExist(t, filepath.Join(root, filepath.FromSlash(tc.memory)))

			// The orientation file wires loading of the index.
			b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.orient)))
			if err != nil {
				t.Fatalf("read orientation: %v", err)
			}
			if !strings.Contains(string(b), tc.directive) {
				t.Errorf("%s missing memory load directive %q", tc.orient, tc.directive)
			}

			// Every agent with a skills dir gets the managing-memory skill.
			if dir := AdapterMust(t, tc.integration).SkillsDir(); dir != "" {
				mustExist(t, filepath.Join(root, filepath.FromSlash(dir), "managing-memory", "SKILL.md"))
			}
		})
	}
}

// [agent memory] re-init must never clobber accumulated memory.
//
// SPEC: story.init.projection
func TestInitPreservesExistingMemory(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, coreassets.FS, Options{Integration: "claude"}); err != nil {
		t.Fatal(err)
	}
	mem := filepath.Join(root, ".claude", "memory", "MEMORY.md")
	accumulated := "# Project memory\n\n- [auth](auth.md) — token lives in the keyring\n"
	if err := os.WriteFile(mem, []byte(accumulated), 0o644); err != nil {
		t.Fatal(err)
	}
	// Re-init with --force (which overwrites generated files) must leave memory alone.
	if _, err := Init(root, coreassets.FS, Options{Integration: "claude", Force: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(mem)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != accumulated {
		t.Errorf("re-init clobbered accumulated memory:\ngot:  %q\nwant: %q", got, accumulated)
	}
}

// AdapterMust returns the adapter for an integration id, failing the test if absent.
func AdapterMust(t *testing.T, id string) Adapter {
	t.Helper()
	a, ok := AdapterFor(id)
	if !ok {
		t.Fatalf("no adapter for %q", id)
	}
	return a
}
