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

	// --with openapi: the contract-first feature. It overwrites main.go / greeting.go
	// / greeting_test.go with the oapi-codegen (strict-server) wiring, adds the
	// contract + codegen config, and carries the go-get + generate scripts. Members
	// import the generated api package by full module path, so the render needs Module.
	feat, ok := m.Features["openapi"]
	if !ok || len(feat.Scripts) == 0 {
		t.Fatalf("go-service missing the openapi feature with codegen scripts: %+v", m.Features)
	}
	oapiDir := t.TempDir()
	fdata := Data{Name: "daemon", Dir: "cmd/daemon", Module: "example.com/acme", Features: map[string]bool{"openapi": true}}
	if _, err := Render(sub, oapiDir, fdata); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderFeature(sub, feat, oapiDir, fdata); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"openapi.yaml", "oapi-codegen.yaml", "main.go", "greeting.go", "greeting_test.go"} {
		if _, err := os.Stat(filepath.Join(oapiDir, p)); err != nil {
			t.Errorf("openapi feature missing %s: %v", p, err)
		}
	}
	// the generated api package is imported by full module path ({{.Module}}/{{.Dir}}).
	wantImport := `"example.com/acme/cmd/daemon/internal/api"`
	mainGo, _ := os.ReadFile(filepath.Join(oapiDir, "main.go"))
	for _, want := range []string{wantImport, "api.HandlerFromMux", "api.NewStrictHandler"} {
		if !strings.Contains(string(mainGo), want) {
			t.Errorf("openapi main.go missing %q:\n%s", want, mainGo)
		}
	}
	greetGo, _ := os.ReadFile(filepath.Join(oapiDir, "greeting.go"))
	for _, want := range []string{wantImport, "api.GreetRequestObject", "api.Greet200JSONResponse"} {
		if !strings.Contains(string(greetGo), want) {
			t.Errorf("openapi greeting.go missing %q:\n%s", want, greetGo)
		}
	}
	// the contract carries the operation + the x-spec trace; oapi-codegen config
	// targets the strict server.
	oapiYaml, _ := os.ReadFile(filepath.Join(oapiDir, "openapi.yaml"))
	for _, want := range []string{"operationId: greet", "x-spec: story.greeting.greet", "title: daemon API"} {
		if !strings.Contains(string(oapiYaml), want) {
			t.Errorf("openapi.yaml missing %q:\n%s", want, oapiYaml)
		}
	}
	if cfg, _ := os.ReadFile(filepath.Join(oapiDir, "oapi-codegen.yaml")); !strings.Contains(string(cfg), "strict-server: true") {
		t.Errorf("oapi-codegen.yaml must enable strict-server:\n%s", cfg)
	}
	// the conditional mise generate task renders only when openapi is selected.
	if mise, _ := os.ReadFile(filepath.Join(oapiDir, "mise.toml")); !strings.Contains(string(mise), "[tasks.generate]") {
		t.Errorf("openapi mise.toml missing the generate task:\n%s", mise)
	}
	// ...and is absent from the default (no-feature) render.
	if mise, _ := os.ReadFile(filepath.Join(dir, "mise.toml")); strings.Contains(string(mise), "[tasks.generate]") {
		t.Errorf("default mise.toml must not carry the generate task:\n%s", mise)
	}
}
