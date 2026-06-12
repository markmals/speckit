package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAndTarget(t *testing.T) {
	src := os.DirFS("testdata/fixture")
	m, err := LoadManifest(src)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "fixture" || m.Install != "echo installed" {
		t.Errorf("manifest = %+v", m)
	}

	data := Data{Name: "My App", Dir: "apps/web"}
	dest := t.TempDir()
	written, err := Render(src, dest, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) == 0 {
		t.Fatal("nothing written")
	}

	// .tmpl rendered + suffix stripped
	if pkg := read(t, filepath.Join(dest, "package.json")); !strings.Contains(pkg, `"name": "My App"`) {
		t.Errorf("package.json = %q", pkg)
	}
	// verbatim copy
	if read(t, filepath.Join(dest, "README.md")) != "# A scaffold\n" {
		t.Error("README not copied verbatim")
	}
	// nested file + a FuncMap helper
	if app := read(t, filepath.Join(dest, "src/app.ts")); !strings.Contains(app, `"my-app"`) {
		t.Errorf("kebab helper: src/app.ts = %q", app)
	}
	// the .tmpl suffix must not leak
	if _, err := os.Stat(filepath.Join(dest, "package.json.tmpl")); !os.IsNotExist(err) {
		t.Error(".tmpl suffix not stripped")
	}

	rt, err := RenderTarget(m, data)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Report != "apps/web/junit.xml" || rt.Source != "apps/web/app" || rt.Format != "junit" {
		t.Errorf("RenderTarget = %+v", rt)
	}
}

func TestCasingHelpers(t *testing.T) {
	for _, c := range []struct{ in, kebab, pascal, camel string }{
		{"My App", "my-app", "MyApp", "myApp"},
		{"consumer-web", "consumer-web", "ConsumerWeb", "consumerWeb"},
		{"FooBar", "foo-bar", "FooBar", "fooBar"},
	} {
		if got := kebab(c.in); got != c.kebab {
			t.Errorf("kebab(%q) = %q, want %q", c.in, got, c.kebab)
		}
		if got := pascal(c.in); got != c.pascal {
			t.Errorf("pascal(%q) = %q, want %q", c.in, got, c.pascal)
		}
		if got := camel(c.in); got != c.camel {
			t.Errorf("camel(%q) = %q, want %q", c.in, got, c.camel)
		}
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
