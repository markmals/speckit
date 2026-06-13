package project

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// writeRuntime populates the .speckit/ runtime the projected commands reference:
// the constitution, the spec/plan/tasks/checklist templates, and a minimal
// extensions manifest. No scripts and no workflow engine (D2) — command logic
// lives in the binary.
//
// SPEC: story.init.projection (scenario.init.projection.fork-divergence)
func writeRuntime(root string, assets fs.FS) ([]string, error) {
	var written []string

	w, err := writeFile(filepath.Join(root, ".speckit", "memory", "constitution.md"),
		rewriteDotdir(mustAsset(assets, "templates/constitution-template.md")))
	if err != nil {
		return nil, err
	}
	written = append(written, w)

	for _, name := range []string{"spec-template.md", "plan-template.md", "tasks-template.md", "checklist-template.md"} {
		b, err := fs.ReadFile(assets, "templates/"+name)
		if err != nil {
			return nil, err
		}
		w, err := writeFile(filepath.Join(root, ".speckit", "templates", name), rewriteDotdir(b))
		if err != nil {
			return nil, err
		}
		written = append(written, w)
	}

	// Minimal extensions manifest — the commands tolerate an empty/missing one.
	w, err = writeFile(filepath.Join(root, ".speckit", "extensions.yml"), []byte("hooks: {}\n"))
	if err != nil {
		return nil, err
	}
	written = append(written, w)

	return written, nil
}

// rewriteDotdir rewrites upstream's .specify/ to the fork's .speckit/ (D6).
func rewriteDotdir(b []byte) []byte {
	return []byte(strings.ReplaceAll(string(b), ".specify/", ".speckit/"))
}

func mustAsset(assets fs.FS, name string) []byte {
	b, _ := fs.ReadFile(assets, name)
	return b
}

func writeFile(path string, data []byte) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// writeFileIfAbsent writes data only when path does not already exist, returning
// "" (and no error) when it leaves an existing file untouched. Used for the
// memory seed: re-running init (even with --force, which clobbers generated
// files) must never overwrite an agent's accumulated memory. A Stat error other
// than "not exist" is returned rather than risking a clobbering write.
func writeFileIfAbsent(path string, data []byte) (string, error) {
	_, err := os.Stat(path)
	if err == nil {
		return "", nil // exists — preserve
	}
	if !os.IsNotExist(err) {
		return "", err // unstattable for some other reason — don't risk overwriting
	}
	return writeFile(path, data)
}
