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

	// First call creates go.mod (no git remote → falls back to the dir base name).
	created, err := ensureRootGoMod(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected ensureRootGoMod to create go.mod")
	}
	b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("go.mod not written: %v", err)
	}
	want := "module " + filepath.Base(dir)
	if got := string(b); !strings.Contains(got, want) || !strings.Contains(got, "go "+goModVersion) {
		t.Errorf("go.mod = %q, want module %q + go %s", got, want, goModVersion)
	}

	// Second call is a no-op — a prior member (or hand-authored module) wins.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/keep\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err = ensureRootGoMod(dir)
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
