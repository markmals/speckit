package main

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/scaffold"
)

// wireMonorepo brings the repo-root mise.toml in line with the current set of
// targets: it merges monorepo_root + the config_roots globs + the union of
// present families' [tools], and — for any family that now has two or more
// members — writes that family's [task_templates] and converts each member's
// still-canonical inline tasks to `extends`. Called after a target is recorded
// (so the new member is counted). Idempotent. A repo whose targets declare no
// family (e.g. only stacks without one) is left untouched.
func wireMonorepo(root string) error {
	cfg, found, err := config.Load(root)
	if err != nil || !found {
		return err
	}

	// Map each target to its family + dir; count members per family.
	type member struct{ dir, family string }
	var members []member
	count := map[string]int{}
	famNames := map[string]bool{}
	var dirs []string
	for _, t := range cfg.Targets {
		if t.Stack == "" {
			continue
		}
		fam, dir, err := familyAndDir(t)
		if err != nil {
			return err
		}
		if fam == "" {
			continue
		}
		members = append(members, member{dir, fam})
		count[fam]++
		famNames[fam] = true
		dirs = append(dirs, dir)
	}
	if len(famNames) == 0 {
		return nil // no family-bearing targets; nothing to wire
	}

	// Load each present family, marking Hoist when it has >=2 members.
	var families []scaffold.Family
	loaded := map[string]scaffold.Family{}
	var names []string
	for n := range famNames {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fam, err := scaffold.LoadFamily(coreassets.FS, n)
		if err != nil {
			return err
		}
		fam.Hoist = count[n] >= 2
		families = append(families, fam)
		loaded[n] = fam
	}

	if _, err := scaffold.EnsureRootMise(root, families, dirs); err != nil {
		return err
	}
	// Promote members of any hoisted family (safe: only canonical tasks convert).
	for _, m := range members {
		fam := loaded[m.family]
		if !fam.Hoist {
			continue
		}
		mise := filepath.Join(root, m.dir, "mise.toml")
		if _, err := scaffold.PromoteMember(mise, fam); err != nil {
			return err
		}
	}
	return nil
}

// familyAndDir resolves a target's family (from its stack's scaffold manifest)
// and member dir (the recorded report's parent — every scaffold's report is
// "<dir>/<file>", so filepath.Dir(t.Report) recovers the member dir, honoring a
// --dir override without re-deriving memberDir/<name>). A stack with no scaffold
// (ts-lib, go-cli) has no family; a malformed scaffold.json is a real error.
func familyAndDir(t config.Target) (family, dir string, err error) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/"+t.Stack)
	if err != nil {
		return "", "", nil // unusable stack path: no family
	}
	m, err := scaffold.LoadManifest(sub)
	if err != nil {
		// fs.Sub is lazy, so a stack without a scaffold dir/json surfaces here as
		// ErrNotExist — that just means "no scaffold, no family". Anything else
		// (e.g. malformed scaffold.json) is a genuine error worth surfacing.
		if errors.Is(err, fs.ErrNotExist) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("stack %q: %w", t.Stack, err)
	}
	return m.Family, filepath.Dir(t.Report), nil
}
