package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".speckit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "specs.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadAbsentIsNotAnError(t *testing.T) {
	_, found, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("absent config should not error: %v", err)
	}
	if found {
		t.Error("found should be false when specs.json is absent")
	}
}

func TestLoadParsesAndDefaults(t *testing.T) {
	root := writeConfig(t, `{
  "version": 2,
  "agent": "claude",
  "targets": {
    "web": { "product": "consumer", "dir": "apps/web", "command": "pnpm -C apps/web test --run", "format": "junit", "report": "apps/web/junit.xml", "source": "apps/web/src" },
    "ios": { "product": "consumer", "format": "swift", "report": "ios.ndjson", "source": "apps/ios/Tests" }
  }
}`)
	cfg, found, err := Load(root)
	if err != nil || !found {
		t.Fatalf("Load: found=%v err=%v", found, err)
	}
	if cfg.Agent != "claude" {
		t.Errorf("agent = %q", cfg.Agent)
	}
	// paths default when omitted
	if cfg.Paths.Specs != "specs" || cfg.Paths.Features != "features" {
		t.Errorf("path defaults = %q/%q", cfg.Paths.Specs, cfg.Paths.Features)
	}
	// a target that omits dir is rooted at the project root
	if cfg.Targets["ios"].Dir != "." {
		t.Errorf("ios dir = %q, want .", cfg.Targets["ios"].Dir)
	}
	if cfg.Targets["web"].Dir != "apps/web" {
		t.Errorf("web dir = %q", cfg.Targets["web"].Dir)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("valid config reported errors: %v", errs)
	}
	if got := len(cfg.ProductTargets()["consumer"]); got != 2 {
		t.Errorf("consumer product targets = %d, want 2", got)
	}
}

func TestSaveNormalizesVersion(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Version: 1, Targets: map[string]Target{
		"web": {Dir: "app", Format: "junit", Report: "j.xml", Source: SourcePaths{"app/src"}},
	}}
	if err := cfg.Save(root); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, File))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Version != SchemaVersion {
		t.Errorf("saved version = %d, want %d", raw.Version, SchemaVersion)
	}
	got, _, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Notice != "" {
		t.Errorf("a freshly saved config must carry no notice, got %q", got.Notice)
	}
}

func TestLoadNoticeForLegacyConfigs(t *testing.T) {
	v1 := writeConfig(t, `{ "version": 1, "targets": { "web": { "format": "junit", "report": "j.xml", "source": "src" } } }`)
	cfg, _, err := Load(v1)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Notice == "" {
		t.Error("a v1 config must carry a notice")
	}

	retired := writeConfig(t, `{
  "version": 2,
  "targets": {
    "web": { "stack": "web", "deploy": { "kind": "x" }, "dir": "app", "format": "junit", "report": "j.xml", "source": "src" }
  }
}`)
	cfg, _, err = Load(retired)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg.Notice, "stack") || !strings.Contains(cfg.Notice, "deploy") {
		t.Errorf("retired keys must be named in the notice, got %q", cfg.Notice)
	}

	current := writeConfig(t, `{ "version": 2, "targets": { "web": { "dir": "app", "format": "junit", "report": "j.xml", "source": "src" } } }`)
	cfg, _, err = Load(current)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Notice != "" {
		t.Errorf("a current config must carry no notice, got %q", cfg.Notice)
	}
}

func TestWorkConfigDefaults(t *testing.T) {
	var c Config
	if w := c.WorkConfig(); w.Provider != WorkMarkdown || w.File != DefaultWorkFile {
		t.Errorf("absent work block = %+v, want markdown on %s", w, DefaultWorkFile)
	}
	c.Work = &Work{Provider: WorkMarkdown}
	if w := c.WorkConfig(); w.File != DefaultWorkFile {
		t.Errorf("markdown with no file = %+v, want %s", w, DefaultWorkFile)
	}
	c.Work = &Work{Provider: WorkNone}
	if w := c.WorkConfig(); w.Provider != WorkNone || w.File != "" {
		t.Errorf("none provider = %+v, want no file default", w)
	}
}

func TestReferenceResolution(t *testing.T) {
	explicit := Config{ReferenceTarget: "ios", Targets: map[string]Target{"web": {}, "ios": {}}}
	if got := explicit.Reference(); got != "ios" {
		t.Errorf("explicit reference = %q, want ios", got)
	}
	sole := Config{Targets: map[string]Target{"web": {}}}
	if got := sole.Reference(); got != "web" {
		t.Errorf("sole-target reference = %q, want web", got)
	}
	multi := Config{Targets: map[string]Target{"web": {}, "ios": {}}}
	if got := multi.Reference(); got != "" {
		t.Errorf("ambiguous reference = %q, want empty", got)
	}
}

func TestValidateCatchesBadTargets(t *testing.T) {
	root := writeConfig(t, `{
  "targets": {
    "web": { "format": "bogus", "report": "j.xml", "source": "src" },
    "api": { "format": "junit", "source": "src" }
  }
}`)
	cfg, _, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	errs := cfg.Validate()
	// web: unknown format; api: missing report → at least 2 problems
	if len(errs) < 2 {
		t.Errorf("expected >=2 validation errors, got %d: %v", len(errs), errs)
	}
}

// the gotest format and the scoped bindings mode validate; an unknown bindings
// mode is rejected.
func TestValidateGoTestAndBindings(t *testing.T) {
	root := writeConfig(t, `{
  "targets": {
    "daemon": { "format": "gotest", "report": "r.json", "source": "cmd", "bindings": "scoped" },
    "web":    { "format": "junit", "report": "j.xml", "source": "app" }
  }
}`)
	cfg, _, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("gotest format + scoped bindings should validate clean, got %v", errs)
	}

	bad := writeConfig(t, `{ "targets": { "x": { "format": "gotest", "report": "r", "source": "s", "bindings": "loose" } } }`)
	cfg, _, err = Load(bad)
	if err != nil {
		t.Fatal(err)
	}
	if errs := cfg.Validate(); len(errs) == 0 {
		t.Error("an unknown bindings mode must be rejected")
	}
}

func TestValidateWorkAndReference(t *testing.T) {
	base := `"targets": { "web": { "dir": ".", "format": "junit", "report": "j.xml", "source": "src" } }`

	unknown := writeConfig(t, `{ `+base+`, "work": { "provider": "jira" } }`)
	cfg, _, _ := Load(unknown)
	if errs := cfg.Validate(); len(errs) == 0 {
		t.Error("an unknown work provider must be rejected")
	}

	noProject := writeConfig(t, `{ `+base+`, "work": { "provider": "github-projects" } }`)
	cfg, _, _ = Load(noProject)
	if errs := cfg.Validate(); len(errs) == 0 {
		t.Error("github-projects with no project number must be rejected")
	}

	badRef := writeConfig(t, `{ `+base+`, "reference_target": "ios" }`)
	cfg, _, _ = Load(badRef)
	if errs := cfg.Validate(); len(errs) == 0 {
		t.Error("a reference_target naming no defined target must be rejected")
	}

	good := writeConfig(t, `{ `+base+`, "reference_target": "web", "work": { "provider": "github-projects", "project": 3 } }`)
	cfg, _, _ = Load(good)
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("a valid reference + work block reported errors: %v", errs)
	}
}

func TestSharedTargetListsBothProducts(t *testing.T) {
	root := writeConfig(t, `{
  "targets": {
    "auth-server": { "products": ["consumer", "admin"], "format": "junit", "report": "r.xml", "source": "services/auth" }
  }
}`)
	cfg, _, _ := Load(root)
	pt := cfg.ProductTargets()
	if len(pt["consumer"]) != 1 || len(pt["admin"]) != 1 {
		t.Errorf("shared target should appear under both products: %v", pt)
	}
}

func TestAddTargetRoundTrips(t *testing.T) {
	root := t.TempDir() // no specs.json yet → AddTarget creates it
	if err := AddTarget(root, "web", Target{Dir: "app", Command: "mise run test", Format: "junit", Report: "j.xml", Source: SourcePaths{"app"}}); err != nil {
		t.Fatal(err)
	}
	cfg, found, err := Load(root)
	if err != nil || !found {
		t.Fatalf("reload: found=%v err=%v", found, err)
	}
	if cfg.Version != SchemaVersion || cfg.Targets["web"].Dir != "app" {
		t.Errorf("cfg = %+v", cfg)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("generated config should validate: %v", errs)
	}
	// a second target preserves the first
	if err := AddTarget(root, "ios", Target{Dir: "ios", Format: "swift", Report: "r", Source: SourcePaths{"s"}}); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ = Load(root)
	if len(cfg.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(cfg.Targets))
	}
}
