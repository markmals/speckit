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
	if m.Stack != "fixture" {
		t.Errorf("manifest = %+v", m)
	}
	if len(m.Scripts) != 1 || len(m.Scripts[0].Commands) != 1 || m.Scripts[0].Commands[0] != "echo installed" {
		t.Errorf("scripts = %+v", m.Scripts)
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

func TestPhasedScripts(t *testing.T) {
	m := Manifest{Scripts: []Script{
		{Phase: 2, Commands: []string{"build"}},
		{Phase: 0, Commands: []string{"pnpm add {{kebab .Name}}-core"}, Silent: true},
		{Phase: 1, Commands: []string{"codegen"}},
	}}
	got, err := m.PhasedScripts(Data{Name: "My App"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Phase != 0 || got[1].Phase != 1 || got[2].Phase != 2 {
		t.Fatalf("phase order = %+v", got)
	}
	// commands are rendered through the template engine (FuncMap + Data)
	if got[0].Commands[0] != "pnpm add my-app-core" {
		t.Errorf("rendered command = %q", got[0].Commands[0])
	}
	if !got[0].Silent {
		t.Error("Silent flag lost in ordering")
	}
	// the source manifest must be untouched (PhasedScripts works on a copy)
	if m.Scripts[1].Commands[0] != "pnpm add {{kebab .Name}}-core" {
		t.Error("PhasedScripts mutated the source manifest")
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

func TestValidateName(t *testing.T) {
	idRule := Manifest{Stack: "swift-package", NameRule: "identifier"}
	// pascal-cases into a letter-led identifier → accepted.
	for _, ok := range []string{"recipe-kit", "greet-tool", "gourmand", "tool-3d", "a", "foo_bar", "Foo"} {
		if err := idRule.ValidateName(ok); err != nil {
			t.Errorf("identifier rule should accept %q: %v", ok, err)
		}
	}
	// pascal-cases to a digit-led (or empty) identifier → rejected.
	for _, bad := range []string{"3d-tool", "9", "123abc", "1"} {
		if err := idRule.ValidateName(bad); err == nil {
			t.Errorf("identifier rule should reject %q (pascal-cases to a digit-led identifier)", bad)
		}
	}
	// the default (no rule) accepts anything the base slug check allowed — incl. a
	// leading digit, harmless for a dir/npm name (a web app or a go-service binary).
	none := Manifest{Stack: "web"}
	for _, any := range []string{"3d-portfolio", "9lives", "web"} {
		if err := none.ValidateName(any); err != nil {
			t.Errorf("default (no) rule should accept %q: %v", any, err)
		}
	}
	// an unknown rule is a manifest error (surfaced at target add).
	if err := (Manifest{Stack: "x", NameRule: "bogus"}).ValidateName("foo"); err == nil {
		t.Error("unknown nameRule should error")
	}
}
