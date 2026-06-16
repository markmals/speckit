package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRawConfig(t *testing.T, root, js string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".speckit"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, File), []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A legacy string source and a new array source both load, and validation passes.
func TestLoadStringAndArraySource(t *testing.T) {
	root := t.TempDir()
	writeRawConfig(t, root, `{
	  "version": 1,
	  "paths": {"specs": "specs", "features": "features"},
	  "targets": {
	    "legacy": {"format": "junit", "report": "r", "source": "apps/web/app"},
	    "multi":  {"format": "gotest", "report": "r2", "source": ["cmd/troved", "internal", "cmd/trove-transcode"]}
	  }
	}`)

	cfg, found, err := Load(root)
	if err != nil || !found {
		t.Fatalf("Load: found=%v err=%v", found, err)
	}
	if got := cfg.Targets["legacy"].Source; len(got) != 1 || got[0] != "apps/web/app" {
		t.Fatalf("legacy string source = %v, want [apps/web/app]", got)
	}
	if got := cfg.Targets["multi"].Source; len(got) != 3 || got[2] != "cmd/trove-transcode" {
		t.Fatalf("array source = %v, want 3 paths", got)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Fatalf("expected valid config, got %v", errs)
	}
}

// Validation rejects an empty array source.
func TestValidateRejectsEmptyArraySource(t *testing.T) {
	root := t.TempDir()
	writeRawConfig(t, root, `{
	  "version": 1,
	  "targets": {"bad": {"format": "junit", "report": "r", "source": []}}
	}`)
	cfg, _, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if errs := cfg.Validate(); len(errs) == 0 {
		t.Error("an empty array source must be a validation error")
	}
}
