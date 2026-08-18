package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// cliCaptureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything fn wrote. The CLI writes results straight to os.Stdout, so
// cobra's SetOut cannot capture them.
func cliCaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stdout = old }()
	fn()
	os.Stdout = old
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return <-done
}

// cliBuildSpecify compiles the specify binary into a fresh temp dir and
// returns its path. ldflags, when non-empty, is passed to the linker — the
// release mechanism injects main.version this way.
func cliBuildSpecify(t *testing.T, ldflags string) string {
	t.Helper()
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not on PATH; cannot build the specify binary")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the cmd/specify package directory")
	}
	bin := filepath.Join(t.TempDir(), "specify")
	args := []string{"build"}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, "-o", bin, ".")
	cmd := exec.Command(goTool, args...)
	cmd.Dir = filepath.Dir(file)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// `specify version` prints the version string and nothing else — no banner —
// and succeeds.
// [scenario.cli.version.plain]
func TestVersionPlain(t *testing.T) {
	var execErr error
	out := cliCaptureStdout(t, func() {
		cmd := rootCmd()
		cmd.SetArgs([]string{"version"})
		execErr = cmd.Execute()
	})
	if execErr != nil {
		t.Fatalf("specify version failed: %v", execErr)
	}
	if out != version+"\n" {
		t.Errorf("specify version printed %q, want exactly %q", out, version+"\n")
	}
}

// `specify version --json` emits a JSON object with a `version` field, and
// --json needs no companion flag.
// [scenario.cli.version.json]
func TestVersionJSON(t *testing.T) {
	var execErr error
	out := cliCaptureStdout(t, func() {
		cmd := rootCmd()
		cmd.SetArgs([]string{"version", "--json"})
		execErr = cmd.Execute()
	})
	if execErr != nil {
		t.Fatalf("specify version --json failed: %v", execErr)
	}
	var got struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if got.Version != version {
		t.Errorf("version field = %q, want %q", got.Version, version)
	}
}

// The reported version is injected at link time, not a hard-coded constant:
// a build with -ldflags "-X main.version=…" reports the injected value.
// [scenario.cli.version.build-injected]
func TestVersionBuildInjected(t *testing.T) {
	bin := cliBuildSpecify(t, "-X main.version=9.9.9-test")
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		t.Fatalf("built binary failed: %v", err)
	}
	if string(out) != "9.9.9-test\n" {
		t.Errorf("injected build reports %q, want %q", out, "9.9.9-test\n")
	}
}

// A build with no injected version reports a clearly-marked development
// version.
// [scenario.cli.version.dev]
func TestVersionDevDefault(t *testing.T) {
	// This test binary is itself a build with no injected version.
	if version != "0.0.0-dev" {
		t.Fatalf("un-injected default version = %q, want %q", version, "0.0.0-dev")
	}
	if !strings.Contains(version, "dev") {
		t.Errorf("default version %q is not clearly marked as a dev build", version)
	}
	out := cliCaptureStdout(t, func() {
		cmd := rootCmd()
		cmd.SetArgs([]string{"version"})
		if err := cmd.Execute(); err != nil {
			t.Errorf("specify version failed: %v", err)
		}
	})
	if out != "0.0.0-dev\n" {
		t.Errorf("un-injected build reports %q, want %q", out, "0.0.0-dev\n")
	}
}
