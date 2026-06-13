package engine

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// sourceExts are the test-source extensions ScanBindings reads — the binding
// formats it understands (Swift Testing traits, Vitest titles) live in these.
var sourceExts = map[string]bool{".swift": true, ".ts": true, ".tsx": true, ".js": true, ".mjs": true}

// walkSourceFiles walks dir and calls fn for each readable file, skipping any
// directory that should never hold spec bindings: .git, node_modules, and every
// directory named in the project's .gitignore files (so the scan never descends
// into generated or vendored trees — notably pnpm's symlink-laden node_modules,
// whose symlinks-to-directories otherwise crash the read). When exts is non-nil
// only files with those extensions are read. A missing dir, or an unreadable
// file (a symlink to a directory, a dangling link, a permission error), is
// skipped rather than fatal.
func walkSourceFiles(dir string, exts map[string]bool, fn func(path string, content []byte)) error {
	skip := ignoredDirNames(dir)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if exts != nil && !exts[filepath.Ext(p)] {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil // unreadable (symlink-to-dir, dangling link, perms) — skip
		}
		fn(p, b)
		return nil
	})
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ignoredDirNames is the set of directory base-names to skip while walking a
// source tree: always .git and node_modules, plus the plain directory entries
// declared in the .gitignore files from dir up to the repo root. It reads a
// pragmatic subset of gitignore syntax — bare names like `node_modules`,
// `dist/`, `/.output` — which is what generated and vendored directories use;
// globs, anchored sub-paths, and negations are left alone (the engine keeps no
// git dependency, so this stays file-based and works offline).
func ignoredDirNames(dir string) map[string]bool {
	skip := map[string]bool{".git": true, "node_modules": true}
	for d := dir; ; {
		readGitignoreDirs(filepath.Join(d, ".gitignore"), skip)
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			break // repo root reached
		}
		parent := filepath.Dir(d)
		if parent == d {
			break // filesystem root reached
		}
		d = parent
	}
	return skip
}

// readGitignoreDirs adds the bare directory-name entries from a .gitignore file
// to skip. Comments, negations, globs, and nested paths are ignored — only a
// plain name (optionally with a leading or trailing slash) becomes a skip.
func readGitignoreDirs(path string, skip map[string]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		line = strings.TrimPrefix(strings.TrimSuffix(line, "/"), "/")
		if line == "" || strings.ContainsAny(line, `/*?[]\`) {
			continue // only bare directory names become skips
		}
		skip[line] = true
	}
}
