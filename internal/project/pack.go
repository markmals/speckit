package project

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// loadPack reads the platform skills for a stack from templates/packs/<stack>/.
// A stack need not ship a pack: library stacks (swift-package, swift-cli) have no
// platform skill suite. So a missing pack dir is only an error when the stack isn't
// a real scaffold either — a known scaffold with no pack returns no skills, which
// keeps `specify packs` from failing on a packless target while still catching a
// genuinely unknown stack name.
func loadPack(assets fs.FS, stack string) ([]Skill, error) {
	dir := "templates/packs/" + stack
	entries, err := fs.ReadDir(assets, dir)
	if errors.Is(err, fs.ErrNotExist) {
		// The scaffold-layout path mirrors cmd/specify (fs.Sub on
		// "templates/scaffolds/<stack>") + scaffold.LoadManifest (reads "scaffold.json");
		// project/ deliberately doesn't import scaffold/, so it's duplicated here. If that
		// layout ever moves, update both or packless stacks start erroring again.
		if _, serr := fs.Stat(assets, "templates/scaffolds/"+stack+"/scaffold.json"); serr == nil {
			return nil, nil // a real scaffold that ships no pack
		}
		return nil, fmt.Errorf("unknown pack %q", stack)
	}
	if err != nil {
		return nil, err
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := fs.ReadFile(assets, dir+"/"+e.Name()+"/SKILL.md")
		if err != nil {
			return nil, err
		}
		skills = append(skills, Skill{Name: e.Name(), Content: string(b)})
	}
	return skills, nil
}

// ProjectPacks projects the platform skills for the given stacks into the
// agent's skills dir (the same dir the process skills go), returning the written
// paths. The agent comes from .speckit/specs.json's `agent`.
func ProjectPacks(root string, assets fs.FS, agentID string, stacks []string) ([]string, error) {
	adapter, ok := AdapterFor(agentID)
	if !ok {
		return nil, fmt.Errorf("unknown agent %q (have: %v)", agentID, AdapterIDs())
	}
	dir := adapter.SkillsDir()
	if dir == "" {
		return nil, fmt.Errorf("agent %q has no skills directory to project packs into", agentID)
	}
	var written []string
	for _, stack := range stacks {
		skills, err := loadPack(assets, stack)
		if err != nil {
			return nil, err
		}
		for _, s := range skills {
			w, err := writeFile(filepath.Join(root, dir, s.Name, "SKILL.md"), []byte(s.Content))
			if err != nil {
				return nil, err
			}
			written = append(written, w)
		}
	}
	return written, nil
}
