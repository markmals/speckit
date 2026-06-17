package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/scaffold"
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
		Report: "apps/web/junit.xml", Source: config.SourcePaths{"apps/web/app"},
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
	// The repo-global dependency-update gate lands at the root (real embedded
	// assets): the deps/check tasks in mise.toml plus renovate.json + the script.
	if !strings.Contains(rootMise, "[tasks.deps]") || !strings.Contains(rootMise, `"npm:renovate" = "latest"`) {
		t.Errorf("root mise.toml missing the deps gate:\n%s", rootMise)
	}
	for _, f := range []string{"renovate.json", "scripts/deps-check.sh"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("deps gate file %s not written to root: %v", f, err)
		}
	}
	// The member must NOT carry the gate — it's repo-global, not per-member.
	if strings.Contains(read(t, filepath.Join(root, "apps/web/mise.toml")), "renovate") {
		t.Error("web member must not carry renovate (gate is repo-global)")
	}

	// --- member 2: apps/web2 (promotion) ---
	writeWebMember(t, root, "apps/web2")
	if err := config.AddTarget(root, "web2", config.Target{
		Stack: "web", Command: "mise //apps/web2:test", Format: "junit",
		Report: "apps/web2/junit.xml", Source: config.SourcePaths{"apps/web2/app"},
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

// renderMember renders a stack's real embedded member tree into root/dir.
func renderMember(t *testing.T, root, stack, name, dir string) {
	t.Helper()
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/"+stack)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Render(sub, filepath.Join(root, dir), scaffold.Data{Name: name, Dir: dir}); err != nil {
		t.Fatal(err)
	}
}

// TestWireMonorepoSwiftCrossStackPromotion registers an apple member then a
// swift-package member and asserts the swift templates hoist and both members
// convert their shared tasks. Apple's tuist build stays inline (not promoted).
func TestWireMonorepoSwiftCrossStackPromotion(t *testing.T) {
	root := t.TempDir()
	mustChdir(t, root)

	// --- member 1: apps/Photos (apple, inline) ---
	renderMember(t, root, "apple", "Photos", "apps/Photos")
	if err := config.AddTarget(root, "Photos", config.Target{
		Stack: "apple", Command: "mise //apps/Photos:test", Format: "swift",
		Report: "apps/Photos/test.swift-events.ndjson", Source: config.SourcePaths{"apps/Photos/Core"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	// one swift member: inline; root has no swift templates and no [tools] tuist
	// (tuist stays in the apple member, not in the swift family contribution).
	rootMise := read(t, filepath.Join(root, "mise.toml"))
	if strings.Contains(rootMise, "[task_templates") {
		t.Errorf("one swift member must stay inline:\n%s", rootMise)
	}

	// --- member 2: packages/Widgets (swift-package, promotion) ---
	renderMember(t, root, "swift-package", "Widgets", "packages/Widgets")
	if err := config.AddTarget(root, "Widgets", config.Target{
		Stack: "swift-package", Command: "mise //packages/Widgets:test", Format: "swift",
		Report: "packages/Widgets/test.swift-events.ndjson", Source: config.SourcePaths{"packages/Widgets/Sources"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	rootMise = read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, `[task_templates."swift:test"]`) {
		t.Errorf("two swift members must hoist swift templates:\n%s", rootMise)
	}
	// both globs present (apps/* and packages/*).
	if !strings.Contains(rootMise, `"apps/*"`) || !strings.Contains(rootMise, `"packages/*"`) {
		t.Errorf("config_roots missing a swift glob:\n%s", rootMise)
	}
	// both members now extend the shared test template.
	for _, d := range []string{"apps/Photos", "packages/Widgets"} {
		m := read(t, filepath.Join(root, d, "mise.toml"))
		if !strings.Contains(m, `extends = "swift:test"`) {
			t.Errorf("%s test not promoted:\n%s", d, m)
		}
	}
	// apple keeps its tuist build inline (not converted to extends = "swift:build").
	ap := read(t, filepath.Join(root, "apps/Photos/mise.toml"))
	if strings.Contains(ap, `extends = "swift:build"`) {
		t.Errorf("apple tuist build must stay inline:\n%s", ap)
	}
}

// TestWireMonorepoGoPromotion renders two go-service members (cmd/api, cmd/worker),
// adds each as a target, wires after each, then asserts the go:test template is
// hoisted to the root config and both members carry extends = "go:test".
func TestWireMonorepoGoPromotion(t *testing.T) {
	root := t.TempDir()
	mustChdir(t, root)
	for _, m := range []struct{ name, dir string }{{"api", "cmd/api"}, {"worker", "cmd/worker"}} {
		renderMember(t, root, "go-service", m.name, m.dir)
		if err := config.AddTarget(root, m.name, config.Target{
			Stack: "go-service", Command: "mise //" + m.dir + ":test", Format: "gotest",
			Report: m.dir + "/test.gotest.json", Source: config.SourcePaths{m.dir}, Bindings: "scoped",
		}); err != nil {
			t.Fatal(err)
		}
		if err := wireMonorepo(root); err != nil {
			t.Fatal(err)
		}
	}
	rootMise := read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, `[task_templates."go:test"]`) {
		t.Errorf("go:test template not hoisted to root:\n%s", rootMise)
	}
	if !strings.Contains(rootMise, `config_roots = ["cmd/*"]`) {
		t.Errorf("config_roots glob wrong (want cmd/*):\n%s", rootMise)
	}
	for _, d := range []string{"cmd/api", "cmd/worker"} {
		if !strings.Contains(read(t, filepath.Join(root, d, "mise.toml")), `extends = "go:test"`) {
			t.Errorf("%s not promoted to extends = \"go:test\"", d)
		}
	}
}
