package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// inlineRun returns the decoded run value of [tasks.<task>] from parsed exprs.
func inlineRun(ex []expr, task string) (string, bool) {
	for i, e := range ex {
		if e.kind.isTable() && e.name == "tasks."+task {
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() && ex[j].name == "run" {
					return ex[j].val, true
				}
			}
		}
	}
	return "", false
}

// TestNodeFamilyMatchesWebInline asserts the node family templates' run strings
// equal the web member scaffold's inline task bodies — the coupling promotion
// relies on. If you change one, change the other.
func TestNodeFamilyMatchesWebInline(t *testing.T) {
	fam, err := LoadFamily(coreassets.FS, "node")
	if err != nil {
		t.Fatal(err)
	}
	sub, _ := fs.Sub(coreassets.FS, "templates/scaffolds/web")
	dir := t.TempDir()
	if _, err := Render(sub, dir, Data{Name: "web", Dir: "apps/web"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	// web member must no longer carry [tools] (hoisted to root).
	if strings.Contains(string(data), "[tools]") {
		t.Errorf("web member must not declare [tools] (hoisted to root):\n%s", data)
	}
	ex, _ := parseExprs(data)
	vars := memberVars(data)
	for task, tpl := range fam.Templates {
		want := substituteVars(tpl.Run, vars)
		got, found := inlineRun(ex, task)
		if !found {
			t.Errorf("web member has no inline [tasks.%s] for family template node:%s", task, task)
			continue
		}
		if got != want {
			t.Errorf("drift: node:%s\n  family:  %q\n  member:  %q", task, want, got)
		}
	}
}

// TestGoFamilyMatchesMemberInline asserts the go family templates' run strings
// equal the go-service member scaffold's inline task bodies — the coupling
// promotion relies on. If you change one, change the other.
func TestGoFamilyMatchesMemberInline(t *testing.T) {
	fam, err := LoadFamily(coreassets.FS, "go")
	if err != nil {
		t.Fatal(err)
	}
	sub, _ := fs.Sub(coreassets.FS, "templates/scaffolds/go-service")
	dir := t.TempDir()
	if _, err := Render(sub, dir, Data{Name: "api", Dir: "cmd/api"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	// go-service member must not carry [tools] (hoisted to root).
	if strings.Contains(string(data), "[tools]") {
		t.Errorf("go-service member must not declare [tools] (hoisted to root):\n%s", data)
	}
	ex, _ := parseExprs(data)
	for _, task := range []string{"dev", "build", "test", "vet", "fmt", "fmt:check"} {
		got, found := inlineRun(ex, task)
		if !found {
			t.Errorf("go-service member has no inline [tasks.%s] for family template go:%s", task, task)
			continue
		}
		if got != fam.Templates[task].Run {
			t.Errorf("drift: go:%s\n  family:  %q\n  member:  %q", task, fam.Templates[task].Run, got)
		}
	}
}

// TestSwiftFamilyMatchesMemberInline asserts the swift family templates' run
// strings (after vars substitution) equal each swift member scaffold's inline
// task bodies — the coupling promotion relies on. If you change one, change
// the other.
func TestSwiftFamilyMatchesMemberInline(t *testing.T) {
	fam, err := LoadFamily(coreassets.FS, "swift")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		stack string
		name  string
		dir   string
		tasks []string
	}{
		{"apple", "Photos", "apps/Photos", []string{"test", "fmt", "lint"}},
		{"swift-package", "Widgets", "packages/Widgets", []string{"test", "build", "fmt", "lint"}},
		{"swift-cli", "Tool", "packages/Tool", []string{"test", "build", "fmt", "lint"}},
	}
	for _, c := range cases {
		sub, _ := fs.Sub(coreassets.FS, "templates/scaffolds/"+c.stack)
		dir := t.TempDir()
		if _, err := Render(sub, dir, Data{Name: c.name, Dir: c.dir}); err != nil {
			t.Fatalf("%s render: %v", c.stack, err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "mise.toml"))
		ex, _ := parseExprs(data)
		vars := memberVars(data)
		for _, task := range c.tasks {
			want := substituteVars(fam.Templates[task].Run, vars)
			got, found := inlineRun(ex, task)
			if !found {
				t.Errorf("%s: no inline [tasks.%s]", c.stack, task)
				continue
			}
			if got != want {
				t.Errorf("drift %s swift:%s\n  family: %q\n  member: %q", c.stack, task, want, got)
			}
		}
	}
}
