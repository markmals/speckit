package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/engine"
)

// cliExecSpecify runs the CLI in-process with the given args and returns the
// command error (nil = exit 0).
func cliExecSpecify(t *testing.T, args ...string) error {
	t.Helper()
	cmd := rootCmd()
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	return cmd.Execute()
}

// cliTreeSnapshot maps every path under root — files to a content hash, dirs
// to a marker — so a before/after diff proves exactly what a command touched.
func cliTreeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			snap[filepath.ToSlash(rel)] = "dir"
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		snap[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

// Registering existing code records the target in .speckit/specs.json and
// touches nothing else on disk: no file rendered, no file modified, and the
// configured test command never runs.
// [scenario.adoption.target-add.registers-existing-code]
func TestTargetAddRegistersExistingCode(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.MkdirAll(filepath.Join("web", "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("web", "src", "app.test.example"), []byte("existing test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before := cliTreeSnapshot(t, root)

	// The command is a sentinel: had target add run any script, RAN would exist.
	err := cliExecSpecify(t, "target", "add", "web",
		"--dir", "web", "--format", "junit", "--report", "web/junit.xml",
		"--source", "web", "--command", "touch RAN", "--bindings", "scoped")
	if err != nil {
		t.Fatalf("target add failed: %v", err)
	}

	cfg, found, err := config.Load(".")
	if err != nil || !found {
		t.Fatalf("config not written: found=%v err=%v", found, err)
	}
	tgt, ok := cfg.Targets["web"]
	if !ok {
		t.Fatalf("target %q not recorded; targets = %v", "web", cfg.TargetNames())
	}
	if tgt.Dir != "web" || tgt.Format != "junit" || tgt.Report != "web/junit.xml" ||
		tgt.Command != "touch RAN" || tgt.Bindings != "scoped" ||
		len(tgt.Source) != 1 || tgt.Source[0] != "web" {
		t.Errorf("recorded target = %+v", tgt)
	}

	after := cliTreeSnapshot(t, root)
	allowed := map[string]bool{".speckit": true, ".speckit/specs.json": true}
	for p, h := range after {
		if allowed[p] {
			continue
		}
		switch bh, ok := before[p]; {
		case !ok:
			t.Errorf("target add created %s — adoption must touch only the config", p)
		case bh != h:
			t.Errorf("target add modified %s — adoption must touch only the config", p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			t.Errorf("target add removed %s", p)
		}
	}
	if _, ok := after[".speckit/specs.json"]; !ok {
		t.Error(".speckit/specs.json was not written")
	}
	if _, err := os.Stat("RAN"); !os.IsNotExist(err) {
		t.Error("target add ran the configured test command")
	}
}

// A --dir that does not exist fails naming the missing directory, and nothing
// is written.
// [scenario.adoption.target-add.requires-existing-dir]
func TestTargetAddRequiresExistingDir(t *testing.T) {
	t.Chdir(t.TempDir())
	err := cliExecSpecify(t, "target", "add", "web",
		"--dir", "missing-dir", "--format", "junit", "--report", "r.xml", "--source", "src")
	if err == nil {
		t.Fatal("target add must fail when --dir does not exist")
	}
	if !strings.Contains(err.Error(), "missing-dir") {
		t.Errorf("error must name the missing directory: %q", err)
	}
	if _, statErr := os.Stat(".speckit"); !os.IsNotExist(statErr) {
		t.Error("target add must write nothing when --dir does not exist")
	}
}

// No help text or error anywhere in the command tree asks the user to choose
// a platform, stack, or scaffold. The sole allowed occurrence is the root
// command's Short — "SpecKit — a stack-agnostic spec engine" — matched
// exactly.
// [scenario.adoption.target-add.no-platform-vocabulary]
func TestTargetAddNoPlatformVocabulary(t *testing.T) {
	forbidden := regexp.MustCompile(`(?i)stack|scaffold|platform`)
	const allowedRootShort = "SpecKit — a stack-agnostic spec engine"

	root := rootCmd()
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		check := func(field, text string) {
			if c == root && field == "Short" && text == allowedRootShort {
				return // the one deliberate occurrence, matched exactly
			}
			if forbidden.MatchString(text) {
				t.Errorf("%s %s speaks platform vocabulary: %q", path, field, text)
			}
		}
		check("Use", c.Use)
		check("Short", c.Short)
		check("Long", c.Long)
		check("Example", c.Example)
		check("flags", c.Flags().FlagUsages())
		check("persistent flags", c.PersistentFlags().FlagUsages())
		for _, sub := range c.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(root, "specify")

	// And the invalid-invocation error paths of target add: each fails naming
	// the offending flag, with no platform vocabulary.
	t.Chdir(t.TempDir())
	cases := []struct {
		flag string
		args []string
	}{
		{"--dir", []string{"target", "add", "web", "--format", "junit", "--report", "r.xml", "--source", "src"}},
		{"--format", []string{"target", "add", "web", "--dir", ".", "--report", "r.xml", "--source", "src"}},
		{"--report", []string{"target", "add", "web", "--dir", ".", "--format", "junit", "--source", "src"}},
		{"--source", []string{"target", "add", "web", "--dir", ".", "--format", "junit", "--report", "r.xml"}},
		{"--format", []string{"target", "add", "web", "--dir", ".", "--format", "tap", "--report", "r.xml", "--source", "src"}},
		{"--bindings", []string{"target", "add", "web", "--dir", ".", "--format", "junit", "--report", "r.xml", "--source", "src", "--bindings", "loose"}},
	}
	for _, tc := range cases {
		err := cliExecSpecify(t, tc.args...)
		if err == nil {
			t.Errorf("%v: expected failure", tc.args)
			continue
		}
		if !strings.Contains(err.Error(), tc.flag) {
			t.Errorf("%v: error must name %s: %q", tc.args, tc.flag, err)
		}
		if forbidden.MatchString(err.Error()) {
			t.Errorf("%v: error speaks platform vocabulary: %q", tc.args, err)
		}
	}
}

// A --format outside the known report formats fails listing every known
// format, and nothing is written.
// [scenario.adoption.target-add.rejects-unknown-format]
func TestTargetAddRejectsUnknownFormat(t *testing.T) {
	t.Chdir(t.TempDir())
	err := cliExecSpecify(t, "target", "add", "web",
		"--dir", ".", "--format", "tap", "--report", "r.xml", "--source", "src")
	if err == nil {
		t.Fatal("target add must reject an unknown --format")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"tap"`) {
		t.Errorf("error must name the rejected format: %q", msg)
	}
	for _, f := range config.Formats {
		if !strings.Contains(msg, f) {
			t.Errorf("error must list known format %q: %q", f, msg)
		}
	}
	if _, statErr := os.Stat(".speckit"); !os.IsNotExist(statErr) {
		t.Error("target add must write nothing on a rejected format")
	}
}

// Repeated --source records every path, and the engine scans each of them for
// bindings — a regression that keeps only the first source misses the binding
// planted in the second.
// [scenario.adoption.target-add.multi-source]
func TestTargetAddMultiSource(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, d := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A binding fixture in the SECOND source dir. The tag is split in this
	// source literal so it never registers as a binding of this file.
	fixture := "package beta\n\nimport \"testing\"\n\n// [scen" + "ario.demo.sample]\nfunc TestBetaSample(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join("beta", "beta_test.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	err := cliExecSpecify(t, "target", "add", "svc",
		"--dir", ".", "--format", "gotest", "--report", "report.json",
		"--source", "alpha", "--source", "beta")
	if err != nil {
		t.Fatalf("target add failed: %v", err)
	}

	cfg, found, err := config.Load(".")
	if err != nil || !found {
		t.Fatalf("config not written: found=%v err=%v", found, err)
	}
	src := cfg.Targets["svc"].Source
	if len(src) != 2 || src[0] != "alpha" || src[1] != "beta" {
		t.Fatalf("recorded sources = %v, want [alpha beta]", src)
	}

	bindings, err := engine.ScanBindingsMany(".", src)
	if err != nil {
		t.Fatal(err)
	}
	foundBinding := false
	for _, b := range bindings {
		if b.Identity == "TestBetaSample" && string(b.Scenario) == "scenario.demo.sample" &&
			strings.HasPrefix(b.File, "beta/") {
			foundBinding = true
		}
	}
	if !foundBinding {
		t.Errorf("engine did not scan the second --source for bindings; got %v", bindings)
	}
}
