package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// SPEC: story.init.projection (scenario.init.projection.fork-divergence)
func TestInitRuntime(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, coreassets.FS, Options{Integration: "claude"}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		".speckit/memory/constitution.md",
		".speckit/templates/spec-template.md",
		".speckit/templates/plan-template.md",
		".speckit/templates/tasks-template.md",
		".speckit/templates/checklist-template.md",
		".speckit/extensions.yml",
	} {
		mustExist(t, filepath.Join(root, p))
	}
	b, _ := os.ReadFile(filepath.Join(root, ".speckit", "memory", "constitution.md"))
	if strings.Contains(string(b), ".specify/") {
		t.Error("runtime still references .specify/ (D6)")
	}
}
