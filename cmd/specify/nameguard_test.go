package main

import (
	"os"
	"strings"
	"testing"
)

// TestTargetAddRejectsNonIdentifierName: a stack whose manifest sets nameRule
// "identifier" (the swift stacks, apple) must reject a name that pascal-cases to a
// non-identifier (a leading digit) at `target add` — BEFORE rendering any files, so
// the user never gets a half-scaffolded package that won't compile.
func TestTargetAddRejectsNonIdentifierName(t *testing.T) {
	// Hermetic: target add operates on "." — if the guard ever regressed, the render
	// would land in this temp dir, not the repo. (The guard fires before any write.)
	t.Chdir(t.TempDir())

	cmd := rootCmd()
	cmd.SetArgs([]string{"target", "add", "3d-tool", "--stack", "swift-package", "--no-install"})
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("target add must reject a digit-leading name for an identifier-rule stack")
	}
	if !strings.Contains(err.Error(), "identifier") {
		t.Errorf("error should explain the identifier constraint, got: %v", err)
	}
	// The guard fires before placement/render, so nothing is written.
	if _, statErr := os.Stat("packages"); !os.IsNotExist(statErr) {
		t.Error("target add must not render any files when the name guard fails")
	}
}
