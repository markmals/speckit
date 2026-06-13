package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
	for _, p := range []string{"package.json", "mise.toml", "vitest.config.ts", ".oxlintrc.json", ".oxfmtrc.json", "app/lib/greeting.ts", "app/lib/greeting.test.ts"} {
		if _, err := os.Stat(filepath.Join(app, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	// the .tmpl suffix must be stripped
	if _, err := os.Stat(filepath.Join(app, "package.json.tmpl")); !os.IsNotExist(err) {
		t.Error("package.json.tmpl suffix not stripped")
	}
	// the quality CI job calls these standard task names — they must exist.
	mise, err := os.ReadFile(filepath.Join(app, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range []string{"[tasks.lint]", `[tasks."fmt:check"]`, "[tasks.typecheck]"} {
		if !strings.Contains(string(mise), task) {
			t.Errorf("mise.toml missing quality task %s", task)
		}
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

	// The github/ subtree drops a project-root CI workflow. Its one GitHub
	// expression (${{ github.ref }}) must survive Go text/template intact, and
	// the scaffold vars must be substituted.
	proj := t.TempDir()
	gh, err := RenderGitHub(sub, proj, data)
	if err != nil {
		t.Fatal(err)
	}
	ciPath := filepath.Join(proj, ".github/workflows/ci.yml")
	ci, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("RenderGitHub did not write ci.yml: %v (wrote %v)", err, gh)
	}
	for _, want := range []string{"target: web", "working_directory: apps/web", "group: ci-${{ github.ref }}", "fmt:check"} {
		if !strings.Contains(string(ci), want) {
			t.Errorf("ci.yml missing %q\n%s", want, ci)
		}
	}
	// the Go-template escape artifact (`{{ "${{" }}`) must be fully resolved —
	// only the GitHub expression ${{ github.ref }} should remain.
	if strings.Contains(string(ci), `{{ "`) || strings.Contains(string(ci), `.Name`) {
		t.Errorf("ci.yml has unrendered template syntax:\n%s", ci)
	}

	// Pillar 2: the github/ subtree also drops the defect-intake surface. Every
	// file must land at its double-nested .github/ path.
	for _, p := range []string{
		".github/PULL_REQUEST_TEMPLATE.md",
		".github/ISSUE_TEMPLATE/defect.yml",
		".github/ISSUE_TEMPLATE/config.yml",
		".github/CODEOWNERS",
		".github/dependabot.yml",
	} {
		if _, err := os.Stat(filepath.Join(proj, filepath.FromSlash(p))); err != nil {
			t.Errorf("RenderGitHub did not write %s: %v", p, err)
		}
	}
	// the defect form stamps the Bug issue type; CODEOWNERS gates the spec library.
	defect, _ := os.ReadFile(filepath.Join(proj, ".github/ISSUE_TEMPLATE/defect.yml"))
	if !strings.Contains(string(defect), "type: Bug") {
		t.Errorf("defect.yml missing `type: Bug`:\n%s", defect)
	}
	owners, _ := os.ReadFile(filepath.Join(proj, ".github/CODEOWNERS"))
	if !strings.Contains(string(owners), "/features/") || !strings.Contains(string(owners), "/specs/") {
		t.Errorf("CODEOWNERS must route /features and /specs:\n%s", owners)
	}
	// dependabot.yml is templated: the npm ecosystem points at the app dir.
	dep, _ := os.ReadFile(filepath.Join(proj, ".github/dependabot.yml"))
	if !strings.Contains(string(dep), "directory: /apps/web") {
		t.Errorf("dependabot.yml npm directory not substituted to the app dir:\n%s", dep)
	}
	if strings.Contains(string(dep), "{{") {
		t.Errorf("dependabot.yml has unrendered template syntax:\n%s", dep)
	}
	// a second target must not clobber an existing ci.yml
	if err := os.WriteFile(ciPath, []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderGitHub(sub, proj, data); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(ciPath); string(b) != "sentinel" {
		t.Error("RenderGitHub clobbered an existing ci.yml")
	}
}
