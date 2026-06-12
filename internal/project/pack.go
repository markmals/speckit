package project

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// loadPack reads the platform skills for a stack from templates/packs/<stack>/.
func loadPack(assets fs.FS, stack string) ([]Skill, error) {
	dir := "templates/packs/" + stack
	entries, err := fs.ReadDir(assets, dir)
	if errors.Is(err, fs.ErrNotExist) {
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
