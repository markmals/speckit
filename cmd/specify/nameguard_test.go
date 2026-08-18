package main

import (
	"os"
	"testing"
)

// validTargetName gates `target add`: the name becomes a config key and a path
// fragment, so anything outside [A-Za-z0-9][A-Za-z0-9._-]* is rejected before
// the config is touched.
func TestValidTargetName(t *testing.T) {
	for _, name := range []string{"web", "ios", "go-service", "a.b_c-d", "3d-tool", "X1"} {
		if !validTargetName(name) {
			t.Errorf("validTargetName(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"", "../escape", "-lead", ".lead", "_lead", "has space", "semi;colon", "star*", "new\nline"} {
		if validTargetName(name) {
			t.Errorf("validTargetName(%q) = true, want false", name)
		}
	}
}

func TestTargetAddRejectsUnsafeName(t *testing.T) {
	t.Chdir(t.TempDir())

	cmd := rootCmd()
	cmd.SetArgs([]string{"target", "add", "../escape", "--dir", ".", "--format", "junit", "--report", "junit.xml", "--source", "src"})
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("target add must reject an unsafe name")
	}
	if _, err := os.Stat(".speckit"); !os.IsNotExist(err) {
		t.Error("target add must not write config when the name guard fails")
	}
}
