package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
)

// cliAddTarget registers a minimal valid target under the current directory.
func cliAddTarget(t *testing.T, name string, extra ...string) {
	t.Helper()
	if err := os.MkdirAll(name, 0o755); err != nil {
		t.Fatal(err)
	}
	args := []string{"target", "add", name,
		"--dir", name, "--format", "junit", "--report", name + "/junit.xml", "--source", name}
	args = append(args, extra...)
	if err := cliExecSpecify(t, args...); err != nil {
		t.Fatalf("target add %s failed: %v", name, err)
	}
}

// An explicit reference_target naming a defined target resolves to it — and
// `target add --reference` is what writes the key.
// [scenario.adoption.reference-target.configured]
func TestReferenceConfigured(t *testing.T) {
	t.Chdir(t.TempDir())
	cliAddTarget(t, "alpha")
	cliAddTarget(t, "beta", "--reference")

	// --reference wrote the key to disk.
	raw, err := os.ReadFile(config.File)
	if err != nil {
		t.Fatal(err)
	}
	var onDisk struct {
		ReferenceTarget string `json:"reference_target"`
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatal(err)
	}
	if onDisk.ReferenceTarget != "beta" {
		t.Errorf("target add --reference wrote reference_target = %q, want %q", onDisk.ReferenceTarget, "beta")
	}

	// And the engine reports that target as the reference.
	cfg, _, err := config.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Reference(); got != "beta" {
		t.Errorf("Reference() = %q, want %q", got, "beta")
	}
}

// With exactly one target and no explicit reference_target, that target is
// the reference — nothing else could be.
// [scenario.adoption.reference-target.sole-target-is-reference]
func TestReferenceSoleTargetIsReference(t *testing.T) {
	t.Chdir(t.TempDir())
	cliAddTarget(t, "solo")

	cfg, _, err := config.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReferenceTarget != "" {
		t.Fatalf("no explicit reference_target expected, got %q", cfg.ReferenceTarget)
	}
	if got := cfg.Reference(); got != "solo" {
		t.Errorf("Reference() = %q, want the sole target %q", got, "solo")
	}
}

// Several targets and no reference_target: no target is the reference and the
// engine privileges none.
// [scenario.adoption.reference-target.unset-privileges-nothing]
func TestReferenceUnsetPrivilegesNothing(t *testing.T) {
	t.Chdir(t.TempDir())
	cliAddTarget(t, "alpha")
	cliAddTarget(t, "beta")

	cfg, _, err := config.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReferenceTarget != "" {
		t.Fatalf("no explicit reference_target expected, got %q", cfg.ReferenceTarget)
	}
	if got := cfg.Reference(); got != "" {
		t.Errorf("Reference() = %q, want \"\" — unset must privilege nothing", got)
	}
}

// A reference_target naming an undefined target is reported by `specify scan`
// (the config finding Validate surfaces), naming the offending value.
// [scenario.adoption.reference-target.must-name-a-target]
func TestScanReportsUndefinedReferenceTarget(t *testing.T) {
	cfg := config.Config{
		Version:         config.SchemaVersion,
		ReferenceTarget: "ghost",
		Targets: map[string]config.Target{
			"web": {Dir: "web", Format: "junit", Report: "r.xml", Source: config.SourcePaths{"src"}},
		},
	}
	var named bool
	for _, e := range cfg.Validate() {
		if strings.Contains(e.Error(), `"ghost"`) && strings.Contains(e.Error(), "reference_target") {
			named = true
		}
	}
	if !named {
		t.Errorf("Validate() must report the undefined reference_target by name; got %v", cfg.Validate())
	}

	// The user-facing path: `specify scan` surfaces the finding and exits red.
	bin := cliBuildSpecify(t, "")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".speckit"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
  "version": 2,
  "reference_target": "ghost",
  "targets": {
    "web": {"dir": "web", "format": "junit", "report": "r.xml", "source": "src"}
  }
}`
	if err := os.WriteFile(filepath.Join(root, config.File), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "scan", "--json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err == nil {
		t.Fatal("specify scan must exit non-zero on an undefined reference_target")
	}
	var exitErr *exec.ExitError
	if ok := errors.As(err, &exitErr); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("specify scan: %v", err)
	}
	var report struct {
		Config []string `json:"config"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("scan --json output is not JSON: %v\n%s", err, out)
	}
	named = false
	for _, e := range report.Config {
		if strings.Contains(e, `"ghost"`) && strings.Contains(e, "reference_target") {
			named = true
		}
	}
	if !named {
		t.Errorf("scan config findings must name the undefined reference: %v", report.Config)
	}
}
