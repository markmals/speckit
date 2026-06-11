package project

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

var updateGoldens = flag.Bool("update", false, "rewrite the golden init manifests")

// TestInitGoldenTrees pins the fork's own init output per agent — the durable
// golden-tree contract (D14). Regenerate with:
//
//	go test ./internal/project -run TestInitGoldenTrees -update
//
// SPEC: story.init.projection
func TestInitGoldenTrees(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "copilot", "generic"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			if _, err := Init(root, coreassets.FS, Options{Integration: agent}); err != nil {
				t.Fatal(err)
			}
			got := manifestOf(t, root)
			golden := filepath.Join("testdata", "goldens", "init", agent+".files.txt")

			if *updateGoldens {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden (run: go test ./internal/project -update): %v", err)
			}
			if got != string(want) {
				t.Errorf("init tree for %s drifted from golden:\n--- got ---\n%s--- want ---\n%s", agent, got, want)
			}
		})
	}
}

// manifestOf returns the sorted, slash-separated relative paths of every file
// under root, newline-terminated.
func manifestOf(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	return strings.Join(paths, "\n") + "\n"
}
