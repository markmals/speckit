package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleFromRemote(t *testing.T) {
	cases := map[string]string{
		"https://github.com/markmals/trove.git":   "github.com/markmals/trove",
		"https://github.com/markmals/trove":       "github.com/markmals/trove",
		"git@github.com:markmals/trove.git":       "github.com/markmals/trove",
		"git@github.com:markmals/trove":           "github.com/markmals/trove",
		"ssh://git@github.com/markmals/trove.git": "github.com/markmals/trove",
		"https://gitlab.com/group/sub/repo.git":   "gitlab.com/group/sub/repo",
		"":                                        "",
		"not a url":                               "",
	}
	for in, want := range cases {
		if got := moduleFromRemote(in); got != want {
			t.Errorf("moduleFromRemote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEnsureRootGoMod(t *testing.T) {
	dir := t.TempDir()

	// No go.mod yet → resolveModulePath derives the dir base name, and
	// ensureRootGoMod writes it.
	mp := resolveModulePath(dir)
	if mp != filepath.Base(dir) {
		t.Errorf("resolveModulePath(no go.mod) = %q, want dir base %q", mp, filepath.Base(dir))
	}
	created, err := ensureRootGoMod(dir, mp)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected ensureRootGoMod to create go.mod")
	}
	want := "module " + filepath.Base(dir)
	if b, _ := os.ReadFile(filepath.Join(dir, "go.mod")); !strings.Contains(string(b), want) || !strings.Contains(string(b), "go "+goModVersion) {
		t.Errorf("go.mod = %q, want module %q + go %s", b, want, goModVersion)
	}

	// With a go.mod present, resolveModulePath reads its module line and
	// ensureRootGoMod is a no-op — a prior member (or hand-authored module) wins.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/keep\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveModulePath(dir); got != "example.com/keep" {
		t.Errorf("resolveModulePath(existing) = %q, want example.com/keep", got)
	}
	created, err = ensureRootGoMod(dir, "example.com/should-not-be-written")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("ensureRootGoMod must not recreate an existing go.mod")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "go.mod")); !strings.Contains(string(b), "example.com/keep") {
		t.Error("ensureRootGoMod clobbered an existing go.mod")
	}
}
