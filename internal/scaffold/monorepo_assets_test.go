package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

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
		var got string
		found := false
		for i, e := range ex {
			if e.kind.isTable() && e.name == "tasks."+task {
				for j := i + 1; j < len(ex); j++ {
					if ex[j].kind.isTable() {
						break
					}
					if ex[j].kind.isKeyValue() && ex[j].name == "run" {
						got, found = ex[j].val, true
					}
				}
			}
		}
		if !found {
			t.Errorf("web member has no inline [tasks.%s] for family template node:%s", task, task)
			continue
		}
		if got != want {
			t.Errorf("drift: node:%s\n  family:  %q\n  member:  %q", task, want, got)
		}
	}
}
