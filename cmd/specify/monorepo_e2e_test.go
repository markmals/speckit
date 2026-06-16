package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
)

// TestWireMonorepoInlineThenPromote exercises both paths: one node member stays
// inline (no templates), and a second node member triggers promotion.
func TestWireMonorepoInlineThenPromote(t *testing.T) {
	root := t.TempDir()
	mustChdir(t, root)

	// --- member 1: apps/web (inline) ---
	writeWebMember(t, root, "apps/web")
	if err := config.AddTarget(root, "web", config.Target{
		Stack: "web", Command: "mise //apps/web:test", Format: "junit",
		Report: "apps/web/junit.xml", Source: "apps/web/app",
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	rootMise := read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, "monorepo_root = true") || !strings.Contains(rootMise, `config_roots = ["apps/*"]`) {
		t.Errorf("root config wrong after member 1:\n%s", rootMise)
	}
	if strings.Contains(rootMise, "[task_templates") {
		t.Errorf("one member must not hoist templates:\n%s", rootMise)
	}
	if !strings.Contains(rootMise, `node = "24"`) || !strings.Contains(rootMise, "1password") {
		t.Errorf("root [tools] missing node family pins:\n%s", rootMise)
	}
	if strings.Contains(read(t, filepath.Join(root, "apps/web/mise.toml")), "extends =") {
		t.Error("single member must stay inline (no extends)")
	}

	// --- member 2: apps/web2 (promotion) ---
	writeWebMember(t, root, "apps/web2")
	if err := config.AddTarget(root, "web2", config.Target{
		Stack: "web", Command: "mise //apps/web2:test", Format: "junit",
		Report: "apps/web2/junit.xml", Source: "apps/web2/app",
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	rootMise = read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, `[task_templates."node:test"]`) {
		t.Errorf("two members must hoist node templates:\n%s", rootMise)
	}
	if !strings.Contains(rootMise, `config_roots = ["apps/*"]`) {
		t.Errorf("one apps/* glob must still cover both members:\n%s", rootMise)
	}
	// both members now extend the shared templates.
	for _, d := range []string{"apps/web", "apps/web2"} {
		m := read(t, filepath.Join(root, d, "mise.toml"))
		if !strings.Contains(m, `extends = "node:test"`) {
			t.Errorf("%s not promoted to extends:\n%s", d, m)
		}
	}
}

// writeWebMember renders the real embedded web member into root/dir (mise.toml only
// is needed for this test; reuse the scaffold render in a focused helper).
func writeWebMember(t *testing.T, root, dir string) {
	t.Helper()
	// Render the web scaffold's mise.toml by reading the embedded member file.
	src, err := coreassetsReadMember("web")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, dir, "mise.toml"), src, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustChdir(t *testing.T, dir string) {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) }) //nolint:errcheck
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// coreassetsReadMember returns a stack's embedded files/mise.toml verbatim.
func coreassetsReadMember(stack string) ([]byte, error) {
	return coreassets.FS.ReadFile("templates/scaffolds/" + stack + "/files/mise.toml")
}
