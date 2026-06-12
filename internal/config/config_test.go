package config

import (
	"os"
	"path/filepath"
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
  "version": 1,
  "agent": "claude",
  "targets": {
    "web": { "product": "consumer", "command": "pnpm -C apps/web test --run", "format": "junit", "report": "apps/web/junit.xml", "source": "apps/web/src" },
    "ios": { "product": "consumer", "format": "swift", "report": "ios.ndjson", "source": "apps/ios/Tests" }
  }
}`)
	cfg, found, err := Load(root)
	if err != nil || !found {
		t.Fatalf("Load: found=%v err=%v", found, err)
	}
	if cfg.Version != 1 || cfg.Agent != "claude" {
		t.Errorf("version/agent = %d/%q", cfg.Version, cfg.Agent)
	}
	// paths default when omitted
	if cfg.Paths.Specs != "specs" || cfg.Paths.Features != "features" {
		t.Errorf("path defaults = %q/%q", cfg.Paths.Specs, cfg.Paths.Features)
	}
	if cfg.Targets["web"].Command != "pnpm -C apps/web test --run" {
		t.Errorf("web command = %q", cfg.Targets["web"].Command)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("valid config reported errors: %v", errs)
	}
	pt := cfg.ProductTargets()
	if got := len(pt["consumer"]); got != 2 {
		t.Errorf("consumer product targets = %d, want 2", got)
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
	if err := AddTarget(root, "web", Target{Stack: "web", Command: "mise run test", Format: "junit", Report: "j.xml", Source: "app"}); err != nil {
		t.Fatal(err)
	}
	cfg, found, err := Load(root)
	if err != nil || !found {
		t.Fatalf("reload: found=%v err=%v", found, err)
	}
	if cfg.Version != 1 || cfg.Targets["web"].Stack != "web" {
		t.Errorf("cfg = %+v", cfg)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("generated config should validate: %v", errs)
	}
	// a second target preserves the first
	if err := AddTarget(root, "ios", Target{Stack: "apple", Format: "swift", Report: "r", Source: "s"}); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ = Load(root)
	if len(cfg.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(cfg.Targets))
	}
}
