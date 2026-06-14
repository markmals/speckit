package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestGoServiceScaffold renders the embedded go-service scaffold and asserts the
// polyglot wiring: members land in cmd/ (memberDir), the target uses the gotest
// format + scoped bindings, a runnable service + a scenario-bound test render, and
// the test's bindings match the seeded story's scenario sub-ids (the consistency
// that makes a fresh `verify` green).
func TestGoServiceScaffold(t *testing.T) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/go-service")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "go-service" || m.MemberDir != "cmd" {
		t.Errorf("stack=%q memberDir=%q, want go-service / cmd", m.Stack, m.MemberDir)
	}
	if m.Target.Format != "gotest" || m.Target.Bindings != "scoped" {
		t.Errorf("target format=%q bindings=%q, want gotest / scoped", m.Target.Format, m.Target.Bindings)
	}
	// Members compose into ONE repo-root go.mod (trove's shape), so the stack is
	// shared-module and ships NO per-member go.mod — `target add` creates the root
	// one (see cmd/specify ensureRootGoMod), tested there.
	if !m.SharedModule {
		t.Error("go-service must be sharedModule (members share a root go.mod)")
	}

	dir := t.TempDir()
	data := Data{Name: "daemon", Dir: "cmd/daemon"}
	if _, err := Render(sub, dir, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"main.go", "greeting.go", "greeting_test.go", "mise.toml", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	// the member carries no go.mod of its own — it joins the shared root module.
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); !os.IsNotExist(err) {
		t.Error("a shared-module member must not render its own go.mod")
	}

	// the seeded story (root/) + the test bindings must agree, or a fresh verify
	// would dangle.
	root := t.TempDir()
	if _, err := RenderRoot(sub, root, data); err != nil {
		t.Fatal(err)
	}
	story, _ := os.ReadFile(filepath.Join(root, "features/0001-greeting/stories/greeting.greet.md"))
	test, _ := os.ReadFile(filepath.Join(dir, "greeting_test.go"))
	for _, scen := range []string{"scenario.greeting.greet.hello", "scenario.greeting.greet.defaults-to-world"} {
		if !strings.Contains(string(story), "<!-- id: "+scen+" -->") {
			t.Errorf("seeded story missing scenario sub-id %q", scen)
		}
		if !strings.Contains(string(test), "// ["+scen+"]") {
			t.Errorf("seeded test missing the leading-comment binding for %q", scen)
		}
	}
}
