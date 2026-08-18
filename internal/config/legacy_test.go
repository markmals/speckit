package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// legacyTargetBody is a valid modern target wrapped in an old-schema file.
const legacyTargetBody = `{
  "version": 1,
  "targets": {
    "web": {"dir": "web", "format": "junit", "report": "web/junit.xml", "source": "web/src", "command": "npm test"}
  }
}`

// A config carrying retired per-target keys (stack, deploy) loads fine: the
// keys are ignored and a single one-line notice names them and points at
// MIGRATION.md — never a hard failure.
// [scenario.adoption.legacy-config.retired-keys-ignored]
func TestLegacyRetiredKeysIgnoredWithOneNotice(t *testing.T) {
	root := writeConfig(t, `{
  "version": 1,
  "targets": {
    "web": {"dir": "web", "stack": "react", "deploy": "vercel", "format": "junit", "report": "web/junit.xml", "source": "web/src", "command": "npm test"}
  }
}`)
	cfg, found, err := Load(root)
	if err != nil {
		t.Fatalf("a legacy config must never hard-fail: %v", err)
	}
	if !found {
		t.Fatal("config not found")
	}
	// The retired keys are ignored — the target still loads intact.
	tgt, ok := cfg.Targets["web"]
	if !ok || tgt.Dir != "web" || tgt.Format != "junit" || tgt.Report != "web/junit.xml" {
		t.Errorf("target did not load intact: %+v", tgt)
	}
	if cfg.Notice == "" {
		t.Fatal("retired keys must surface a notice")
	}
	if strings.Contains(strings.TrimSpace(cfg.Notice), "\n") {
		t.Errorf("notice must be a single line: %q", cfg.Notice)
	}
	for _, want := range []string{"stack", "deploy", "MIGRATION.md"} {
		if !strings.Contains(cfg.Notice, want) {
			t.Errorf("notice must mention %q: %q", want, cfg.Notice)
		}
	}
}

// A config declaring an older schema version loads, its targets intact, and
// validates clean — nothing about the old version stops an engine command.
// [scenario.adoption.legacy-config.older-version-loads]
func TestLegacyOlderVersionLoads(t *testing.T) {
	root := writeConfig(t, legacyTargetBody)
	cfg, found, err := Load(root)
	if err != nil {
		t.Fatalf("an old schema version must load: %v", err)
	}
	if !found {
		t.Fatal("config not found")
	}
	tgt, ok := cfg.Targets["web"]
	if !ok || tgt.Format != "junit" || tgt.Command != "npm test" ||
		len(tgt.Source) != 1 || tgt.Source[0] != "web/src" {
		t.Errorf("targets did not survive the old version: %+v", cfg.Targets)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("an old but well-formed config must validate clean, got %v", errs)
	}
	if cfg.Notice == "" {
		t.Error("an older schema version should surface an informational notice")
	}
}

// A command that writes the config lands it at the current schema version.
// AddTarget is the write path `specify target add` uses.
// [scenario.adoption.legacy-config.rewritten-current]
func TestLegacyWriteNormalizesToCurrentVersion(t *testing.T) {
	root := writeConfig(t, legacyTargetBody)
	err := AddTarget(root, "api", Target{
		Dir: "api", Format: "gotest", Report: "api/report.json", Source: SourcePaths{"api"},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, File))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatal(err)
	}
	if onDisk.Version != SchemaVersion {
		t.Errorf("rewritten config is v%d, want current v%d", onDisk.Version, SchemaVersion)
	}
	// The rewrite kept the old target and added the new one.
	cfg, _, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Targets["web"]; !ok {
		t.Error("rewrite dropped the pre-existing target")
	}
	if _, ok := cfg.Targets["api"]; !ok {
		t.Error("rewrite lost the added target")
	}
	if cfg.Notice != "" {
		t.Errorf("a current-version rewrite must not carry a legacy notice: %q", cfg.Notice)
	}
}
