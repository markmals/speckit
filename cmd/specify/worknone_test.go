package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/config"
)

// worknoneWriteConfig writes .speckit/specs.json under root.
func worknoneWriteConfig(t *testing.T, root, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".speckit"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".speckit", "specs.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// worknoneCaptureStdout runs fn with os.Stdout redirected and returns what
// it printed.
func worknoneCaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// Provider "none": every one of the five verbs prints exactly one line
// saying no work provider is configured, and exits 0.
//
// [scenario.work.providers.none-is-quiet]
func TestNoneProviderIsQuietOnEveryVerb(t *testing.T) {
	root := t.TempDir()
	worknoneWriteConfig(t, root, `{"version": 2, "work": {"provider": "none"}}`)
	t.Chdir(root)

	verbs := [][]string{
		{"work", "ready"},
		{"work", "create", "A title"},
		{"work", "claim", "wk-1"},
		{"work", "move", "wk-1", "done"},
		{"work", "list"},
	}
	for _, args := range verbs {
		out := worknoneCaptureStdout(t, func() {
			cmd := rootCmd()
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Errorf("%v must exit 0 with provider none, got %v", args, err)
			}
		})
		if out != "no work provider configured\n" {
			t.Errorf("%v printed %q, want exactly one line saying no work provider is configured", args, out)
		}
	}
}

// An absent work block is not an error: it resolves to the markdown
// provider on WORK.md — through config defaulting and through the CLI's
// own provider resolver, with and without a config file at all.
//
// [scenario.work.providers.markdown-is-default]
func TestMarkdownIsTheDefaultWorkProvider(t *testing.T) {
	w := (config.Config{}).WorkConfig()
	if w.Provider != config.WorkMarkdown || w.File != config.DefaultWorkFile {
		t.Errorf("absent work block resolves to %+v, want markdown on %s", w, config.DefaultWorkFile)
	}

	// No config file at all.
	t.Chdir(t.TempDir())
	p, err := resolveWorkProvider(&cobra.Command{})
	if err != nil {
		t.Fatalf("absent config must not be an error: %v", err)
	}
	if p == nil || p.Name() != config.WorkMarkdown {
		t.Fatalf("provider without any config = %v, want markdown", p)
	}

	// A config file with no work block.
	root := t.TempDir()
	worknoneWriteConfig(t, root, `{"version": 2}`)
	t.Chdir(root)
	p, err = resolveWorkProvider(&cobra.Command{})
	if err != nil {
		t.Fatalf("config without a work block must not be an error: %v", err)
	}
	if p == nil || p.Name() != config.WorkMarkdown {
		t.Fatalf("provider without a work block = %v, want markdown", p)
	}
}

// A provider id outside markdown|beads|github-projects|none is rejected:
// config.Validate — the finding `specify scan` prints for the config — names
// the bad id and the valid vocabulary, and the CLI resolver refuses it.
//
// [scenario.work.providers.unknown-provider-rejected]
func TestUnknownWorkProviderRejected(t *testing.T) {
	cfg := config.Config{Work: &config.Work{Provider: "jira"}}
	var found string
	for _, e := range cfg.Validate() {
		if strings.Contains(e.Error(), `"jira"`) {
			found = e.Error()
		}
	}
	if found == "" {
		t.Fatalf("Validate() = %v, want a finding naming the unknown provider", cfg.Validate())
	}
	for _, want := range config.WorkProviders {
		if !strings.Contains(found, want) {
			t.Errorf("finding %q does not list the valid provider %q", found, want)
		}
	}

	root := t.TempDir()
	worknoneWriteConfig(t, root, `{"version": 2, "work": {"provider": "jira"}}`)
	t.Chdir(root)
	if _, err := resolveWorkProvider(&cobra.Command{}); err == nil || !strings.Contains(err.Error(), `"jira"`) {
		t.Errorf("resolver must reject the unknown provider by name, got %v", err)
	}
}
