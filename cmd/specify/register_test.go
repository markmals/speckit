package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/config"
)

// register an existing member whose stack HAS a scaffold (go-service): the wiring
// is seeded from the manifest — gotest + scoped + the cmd/<name> source — without
// writing any files.
func TestRegisterTargetFromManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd/troved"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := registerTarget(root, regOpts{name: "troved", stack: "go-service", dir: "cmd/troved"}); err != nil {
		t.Fatal(err)
	}
	cfg, found, err := config.Load(root)
	if err != nil || !found {
		t.Fatalf("Load: found=%v err=%v", found, err)
	}
	got := cfg.Targets["troved"]
	if got.Stack != "go-service" || got.Format != "gotest" || got.Bindings != "scoped" || got.Source.First() != "cmd/troved" {
		t.Errorf("target = %+v, want go-service/gotest/scoped/source=cmd/troved", got)
	}
	if got.Report == "" || got.Command == "" {
		t.Errorf("manifest should seed command + report: %+v", got)
	}
	// register writes NO files into the member.
	if entries, _ := os.ReadDir(filepath.Join(root, "cmd/troved")); len(entries) != 0 {
		t.Errorf("register must not scaffold files, found %d", len(entries))
	}
}

// register a member whose stack has NO scaffold (e.g. go-cli): the wiring comes
// entirely from flags.
func TestRegisterTargetExplicitFlags(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "packages/services/src"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := registerTarget(root, regOpts{
		name: "services", stack: "go-cli", dir: "packages/services",
		format: "junit", command: "cd packages/services && mise run test",
		report: "packages/services/junit.xml", source: []string{"packages/services/src"}, bindings: "scoped",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := config.Load(root)
	got := cfg.Targets["services"]
	if got.Stack != "go-cli" || got.Format != "junit" || got.Source.First() != "packages/services/src" || got.Bindings != "scoped" {
		t.Errorf("target = %+v", got)
	}
}

// a flag overrides the manifest default (web has no bindings default → scoped wins).
func TestRegisterTargetFlagOverridesManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps/web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := registerTarget(root, regOpts{name: "web", stack: "web", dir: "apps/web", bindings: "scoped"}); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := config.Load(root)
	if got := cfg.Targets["web"]; got.Bindings != "scoped" || got.Format != "junit" {
		t.Errorf("override not applied: %+v", got)
	}
}

func TestRegisterTargetErrors(t *testing.T) {
	root := t.TempDir()

	// non-existent member dir
	if err := registerTarget(root, regOpts{name: "ghost", stack: "go-service", dir: "cmd/ghost"}); err == nil {
		t.Error("expected error for a non-existent member dir")
	}
	// stack without a scaffold + no wiring flags → incomplete
	if err := os.MkdirAll(filepath.Join(root, "packages/x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := registerTarget(root, regOpts{name: "x", stack: "go-cli", dir: "packages/x"}); err == nil {
		t.Error("expected error for incomplete wiring (no scaffold, no --format)")
	}
	// bad name
	if err := registerTarget(root, regOpts{name: "../escape", stack: "go-service", dir: "cmd/x"}); err == nil {
		t.Error("expected error for an unsafe target name")
	}
}

// register a multi-source target via repeated --source flags: the 3-path array
// round-trips through specs.json. No member files are written (register never
// scaffolds), though the go-service manifest still seeds the test wiring.
func TestRegisterTargetMultiSource(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd/troved"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := registerTarget(root, regOpts{
		name: "go-service", stack: "go-service", dir: "cmd/troved",
		source: []string{"cmd/troved", "internal", "cmd/trove-transcode"},
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := config.Load(root)
	tgt := cfg.Targets["go-service"]
	if len(tgt.Source) != 3 || tgt.Source[0] != "cmd/troved" || tgt.Source[2] != "cmd/trove-transcode" {
		t.Fatalf("expected a 3-source target, got %v", tgt.Source)
	}
	// the --source override is orthogonal to the manifest-seeded wiring
	if tgt.Format != "gotest" {
		t.Errorf("expected format seeded from the go-service manifest, got %q", tgt.Format)
	}
}
