package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestWebScaffold renders the real embedded web scaffold — a guard against a
// broken scaffold.json or an unrenderable template.
func TestWebScaffold(t *testing.T) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/web")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "web" {
		t.Fatalf("stack = %q", m.Stack)
	}
	data := Data{Name: "web", Dir: "apps/web"}

	app := t.TempDir()
	if _, err := Render(sub, app, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"package.json", "mise.toml", "vitest.config.ts", "app/lib/greeting.ts", "app/lib/greeting.test.ts"} {
		if _, err := os.Stat(filepath.Join(app, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	// the .tmpl suffix must be stripped
	if _, err := os.Stat(filepath.Join(app, "package.json.tmpl")); !os.IsNotExist(err) {
		t.Error("package.json.tmpl suffix not stripped")
	}

	root := t.TempDir()
	w, err := RenderRoot(sub, root, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) == 0 {
		t.Fatal("RenderRoot seeded no example feature")
	}
	if _, err := os.Stat(filepath.Join(root, "features/0001-welcome/stories/welcome.greet.md")); err != nil {
		t.Errorf("seeded feature missing: %v", err)
	}

	rt, err := RenderTarget(m, data)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Format != "junit" || rt.Source != "apps/web/app" {
		t.Errorf("RenderTarget = %+v", rt)
	}
}
