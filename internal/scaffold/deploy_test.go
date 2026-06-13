package scaffold

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestRenderDeployAllKinds renders every embedded deploy template: SpecKit's [[ ]]
// vars must be substituted while GitHub ${{ }} expressions survive verbatim.
func TestRenderDeployAllKinds(t *testing.T) {
	kinds, err := fs.ReadDir(coreassets.FS, "templates/deploy")
	if err != nil {
		t.Fatal(err)
	}
	if len(kinds) == 0 {
		t.Fatal("no deploy templates embedded")
	}
	data := Data{Name: "web", Dir: "apps/web"}
	for _, k := range kinds {
		if !k.IsDir() {
			continue
		}
		t.Run(k.Name(), func(t *testing.T) {
			out, err := RenderDeploy(coreassets.FS, k.Name(), data)
			if err != nil {
				t.Fatal(err)
			}
			s := string(out)
			// SpecKit vars resolved — no [[ … ]] left, and the app dir substituted.
			if strings.Contains(s, "[[") || strings.Contains(s, "]]") {
				t.Errorf("unresolved [[ ]] delimiter:\n%s", s)
			}
			if !strings.Contains(s, "apps/web") {
				t.Errorf("app dir not substituted:\n%s", s)
			}
			// GitHub expressions must survive untouched.
			if !strings.Contains(s, "${{") {
				t.Errorf("no ${{ }} survived — GitHub expressions were eaten:\n%s", s)
			}
			if !strings.Contains(s, "name: deploy") && !strings.Contains(s, "jobs:") {
				t.Errorf("doesn't look like a workflow:\n%s", s)
			}
		})
	}
}

func TestRenderDeployUnknownKind(t *testing.T) {
	if _, err := RenderDeploy(coreassets.FS, "heroku", Data{}); err == nil {
		t.Error("expected error for unknown deploy kind")
	}
}
